package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Validate
// ---------------------------------------------------------------------------

func TestValidate_ValidConfig(t *testing.T) {
	t.Parallel()
	cfg := Config{
		WorktreePath: "~/worktrees",
		Repos: []RepoConfig{
			{Name: "alpha", Path: "~/repos/alpha"},
			{Name: "beta", Path: "~/repos/beta"},
		},
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidate_MissingWorktreePath(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Repos: []RepoConfig{
			{Name: "alpha", Path: "~/repos/alpha"},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worktree_path is required")
}

func TestValidate_NoRepos(t *testing.T) {
	t.Parallel()
	cfg := Config{
		WorktreePath: "~/worktrees",
		Repos:        []RepoConfig{},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one repo")
}

func TestValidate_RepoMissingName(t *testing.T) {
	t.Parallel()
	cfg := Config{
		WorktreePath: "~/worktrees",
		Repos: []RepoConfig{
			{Name: "", Path: "~/repos/alpha"},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidate_RepoMissingPath(t *testing.T) {
	t.Parallel()
	cfg := Config{
		WorktreePath: "~/worktrees",
		Repos: []RepoConfig{
			{Name: "alpha", Path: ""},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestValidate_DuplicateRepoName(t *testing.T) {
	t.Parallel()
	cfg := Config{
		WorktreePath: "~/worktrees",
		Repos: []RepoConfig{
			{Name: "alpha", Path: "~/repos/alpha"},
			{Name: "alpha", Path: "~/repos/alpha-copy"},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate name")
}

// ---------------------------------------------------------------------------
// EffectiveWorktreePath
// ---------------------------------------------------------------------------

func TestEffectiveWorktreePath_PerRepoOverride(t *testing.T) {
	t.Parallel()
	rc := RepoConfig{
		Name:         "alpha",
		Path:         "~/repos/alpha",
		WorktreePath: "/custom/worktrees",
	}

	result := rc.EffectiveWorktreePath("~/global/worktrees")
	assert.Equal(t, "/custom/worktrees", result)
}

func TestEffectiveWorktreePath_FallsBackToGlobal(t *testing.T) {
	t.Parallel()
	rc := RepoConfig{
		Name: "alpha",
		Path: "~/repos/alpha",
	}

	result := rc.EffectiveWorktreePath("/global/worktrees")
	assert.Equal(t, "/global/worktrees", result)
}

func TestEffectiveWorktreePath_ExpandsTilde(t *testing.T) {
	t.Parallel()
	rc := RepoConfig{
		Name:         "alpha",
		Path:         "~/repos/alpha",
		WorktreePath: "~/custom/trees",
	}

	result := rc.EffectiveWorktreePath("~/global")
	// Should expand ~/ to home dir. We can't know the exact home, but
	// it should NOT start with "~/" anymore.
	assert.NotEqual(t, "~/custom/trees", result, "tilde should be expanded")
	assert.NotContains(t, result, "~/")
}

func TestEffectiveWorktreePath_GlobalExpandsTilde(t *testing.T) {
	t.Parallel()
	rc := RepoConfig{
		Name: "alpha",
		Path: "~/repos/alpha",
	}

	result := rc.EffectiveWorktreePath("~/global/trees")
	assert.NotContains(t, result, "~/", "global tilde should be expanded")
}

func TestEffectiveWorktreePath_AbsolutePathUnchanged(t *testing.T) {
	t.Parallel()
	rc := RepoConfig{
		Name:         "alpha",
		Path:         "/repos/alpha",
		WorktreePath: "/absolute/path/worktrees",
	}

	result := rc.EffectiveWorktreePath("/global/trees")
	assert.Equal(t, "/absolute/path/worktrees", result)
}

// ---------------------------------------------------------------------------
// expandPath (tested indirectly through EffectiveWorktreePath but also directly)
// ---------------------------------------------------------------------------

func TestExpandPath_NoTilde(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/some/path", expandPath("/some/path"))
}

func TestExpandPath_TildeOnly(t *testing.T) {
	t.Parallel()
	// "~" without trailing "/" should not be expanded.
	assert.Equal(t, "~", expandPath("~"))
}

func TestExpandPath_EmptyString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", expandPath(""))
}

func TestExpandPath_RelativePath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "relative/path", expandPath("relative/path"))
}
