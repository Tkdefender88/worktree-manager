package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Tkdefender88/worktree-manager/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Add the current repository to the worktree manager config",
	Long: `Finds the nearest git repository root (searching upward from the current
directory) and adds it to the worktree manager config file.

If -n/--name is not provided you will be prompted for a project name.
If the config file does not exist yet, you will also be prompted for
the default worktree base path.`,
	SilenceUsage: true,
	RunE:         runInit,
}

var initName string

func init() {
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "project name to use in the config")
}

func runInit(cmd *cobra.Command, args []string) error {
	// 1. Find the nearest git root.
	gitRoot, err := findGitRoot()
	if err != nil {
		return err
	}

	// 2. Compress to ~/... if under $HOME.
	gitRoot = compressPath(gitRoot)

	// 3. Resolve the project name.
	name := initName
	if name == "" {
		name, err = prompt("Project name: ")
		if err != nil {
			return fmt.Errorf("reading project name: %w", err)
		}
	}
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// 4. If worktree_path is not set in the config, prompt for it.
	if viper.GetString("worktree_path") == "" {
		wtp, err := prompt("Default worktree path (e.g. ~/worktrees): ")
		if err != nil {
			return fmt.Errorf("reading worktree path: %w", err)
		}
		if wtp == "" {
			return fmt.Errorf("worktree_path cannot be empty")
		}
		viper.Set("worktree_path", wtp)
	}

	// 5. Load existing repos and check for duplicates.
	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	for _, r := range cfg.Repos {
		if r.Name == name {
			return fmt.Errorf("a repo named %q is already in the config", name)
		}
		if expandPath(r.Path) == expandPath(gitRoot) {
			return fmt.Errorf("path %s is already registered as %q", gitRoot, r.Name)
		}
	}

	// 6. Append the new repo and write the config.
	newRepo := config.RepoConfig{
		Name: name,
		Path: gitRoot,
	}
	cfg.Repos = append(cfg.Repos, newRepo)

	// Convert to a plain []map[string]any so Viper writes it correctly as a
	// YAML sequence.
	repoMaps := make([]map[string]any, len(cfg.Repos))
	for i, r := range cfg.Repos {
		m := map[string]any{
			"name": r.Name,
			"path": r.Path,
		}
		if r.WorktreePath != "" {
			m["worktree_path"] = r.WorktreePath
		}
		repoMaps[i] = m
	}
	viper.Set("repos", repoMaps)

	// Determine the config file path to write to.
	cfgPath := viper.ConfigFileUsed()
	if cfgPath == "" {
		// Config file didn't exist yet — use the default location.
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		cfgPath = filepath.Join(home, ".config", "worktree-manager", "config.yaml")
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := viper.WriteConfigAs(cfgPath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Added %q (%s) to %s\n", name, gitRoot, cfgPath)
	return nil
}

// findGitRoot walks up from the current working directory until it finds a
// directory containing a .git entry, or returns an error if none is found.
func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root without finding .git.
			return "", fmt.Errorf("no git repository found in current directory or any parent")
		}
		dir = parent
	}
}

// compressPath replaces the user's home directory prefix with ~.
func compressPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	// Ensure home has no trailing slash for clean prefix matching.
	home = filepath.Clean(home)
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + string(filepath.Separator) + path[len(home)+1:]
	}
	if path == home {
		return "~"
	}
	return path
}

// prompt prints msg to stdout and returns the trimmed line read from stdin.
func prompt(msg string) (string, error) {
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
