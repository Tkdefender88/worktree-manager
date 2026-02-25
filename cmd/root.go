// Package cmd implements the CLI layer for the worktree manager using Cobra and Viper.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Tkdefender88/worktree-manager/internal/config"
	"github.com/Tkdefender88/worktree-manager/internal/git"
	"github.com/Tkdefender88/worktree-manager/internal/tui"
	"github.com/Tkdefender88/worktree-manager/worktree"
)

var (
	cfgFile string
	debug   bool
)

var rootCmd = &cobra.Command{
	Use:   "worktree-manager",
	Short: "A TUI for managing git worktrees across multiple repositories",
	Long: `Worktree Manager is a terminal UI for managing git worktrees
across multiple repositories. Configure your repos in a YAML config file
and use the TUI to list, create, delete, and switch between worktrees.`,
	SilenceUsage: true,
	RunE:         run,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.config/worktree-manager/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false,
		"dump all tea.Msg values to debug.log")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".config", "worktree-manager")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// It's fine if the config file doesn't exist yet — we'll handle that in run().
	_ = viper.ReadInConfig()
}

func run(cmd *cobra.Command, args []string) error {
	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w\n\nCreate a config file at ~/.config/worktree-manager/config.yaml", err)
	}

	// Build worktree.Repo slice from config.
	repos := make([]worktree.Repo, len(cfg.Repos))
	for i, rc := range cfg.Repos {
		repos[i] = worktree.Repo{
			Name:         rc.Name,
			Path:         expandPath(rc.Path),
			WorktreePath: rc.EffectiveWorktreePath(cfg.WorktreePath),
		}
	}

	// Build the worktree component.
	svc := git.NewWorktreeService()
	wtModel := worktree.New(svc,
		worktree.WithRepos(repos),
		worktree.WithDefaultWorktreePath(expandPath(cfg.WorktreePath)),
	)

	// Wrap it in the root TUI model (composition pattern).
	var tuiOpts []tui.Option
	if debug {
		f, err := os.Create("debug.log")
		if err != nil {
			return fmt.Errorf("creating debug.log: %w", err)
		}
		defer f.Close()
		tuiOpts = append(tuiOpts, tui.WithDebug(f))
		fmt.Fprintln(os.Stderr, "Debug mode enabled, writing to debug.log")
	}

	rootModel := tui.New(wtModel, tuiOpts...)
	p := tea.NewProgram(rootModel, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	// If the user switched to a worktree, print the path so callers can cd.
	if m, ok := finalModel.(tui.Model); ok {
		if output := m.FinalOutput(); output != "" {
			fmt.Println(output)
		}
	}

	return nil
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
