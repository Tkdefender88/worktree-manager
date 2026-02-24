// Package config defines the configuration structures for the worktree manager.
// These are used by the CLI layer (Cobra/Viper) to parse config files and
// translate them into the options the worktree TUI model expects.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the top-level configuration loaded from the config file.
type Config struct {
	// WorktreePath is the default base directory where worktrees are created.
	// Structure: <worktree_path>/<repo-name>/<branch-name>
	WorktreePath string       `mapstructure:"worktree_path"`
	Repos        []RepoConfig `mapstructure:"repos"`
}

// RepoConfig represents a single git repository to manage.
type RepoConfig struct {
	// Name is the display name for this repository.
	Name string `mapstructure:"name"`
	// Path is the filesystem path to the main/bare repository.
	Path string `mapstructure:"path"`
	// WorktreePath optionally overrides the global WorktreePath for this repo.
	WorktreePath string `mapstructure:"worktree_path"`
}

// EffectiveWorktreePath returns the worktree base path for this repo,
// falling back to the global default if no per-repo override is set.
func (r RepoConfig) EffectiveWorktreePath(globalDefault string) string {
	if r.WorktreePath != "" {
		return expandPath(r.WorktreePath)
	}
	return expandPath(globalDefault)
}

// Validate checks the configuration for errors.
func (c Config) Validate() error {
	if c.WorktreePath == "" {
		return fmt.Errorf("worktree_path is required")
	}
	if len(c.Repos) == 0 {
		return fmt.Errorf("at least one repo must be configured")
	}
	seen := make(map[string]bool)
	for i, r := range c.Repos {
		if r.Name == "" {
			return fmt.Errorf("repos[%d]: name is required", i)
		}
		if r.Path == "" {
			return fmt.Errorf("repos[%d] (%s): path is required", i, r.Name)
		}
		if seen[r.Name] {
			return fmt.Errorf("repos[%d]: duplicate name %q", i, r.Name)
		}
		seen[r.Name] = true
	}
	return nil
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
