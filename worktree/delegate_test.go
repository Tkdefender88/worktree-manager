package worktree

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "main",
			maxLen:   12,
			expected: "main",
		},
		{
			name:     "exact length unchanged",
			input:    "feature-auth",
			maxLen:   12,
			expected: "feature-auth",
		},
		{
			name:     "long string truncated",
			input:    "feature-auth-login-flow",
			maxLen:   12,
			expected: "feature-auth…",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   12,
			expected: "",
		},
		{
			name:     "single char limit",
			input:    "abc",
			maxLen:   1,
			expected: "a…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, truncate(tt.input, tt.maxLen))
		})
	}
}

// ---------------------------------------------------------------------------
// branchColumn
// ---------------------------------------------------------------------------

func TestBranchColumn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		branch string
		maxLen int
		// We verify the rune-length of the output is always maxLen+1
		// (the column width including ellipsis slot).
	}{
		{
			name:   "short branch is padded",
			branch: "main",
			maxLen: 12,
		},
		{
			name:   "exact length branch",
			branch: "feature-auth",
			maxLen: 12,
		},
		{
			name:   "long branch is truncated",
			branch: "feature-auth-login-flow",
			maxLen: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := branchColumn(tt.branch, tt.maxLen)
			runes := []rune(result)
			assert.Equal(t, tt.maxLen+1, len(runes),
				"branchColumn should produce %d-rune output, got %d: %q",
				tt.maxLen+1, len(runes), result)
		})
	}
}

// ---------------------------------------------------------------------------
// shortPath
// ---------------------------------------------------------------------------

func TestShortPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		wtPath   string
		repo     Repo
		expected string
	}{
		{
			name:     "relative to worktree base",
			wtPath:   "/worktrees/alpha/feature",
			repo:     Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
			expected: "feature",
		},
		{
			name:     "relative to repo parent (main checkout)",
			wtPath:   "/repos/alpha",
			repo:     Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
			expected: "alpha",
		},
		{
			name:     "no worktree path configured",
			wtPath:   "/repos/alpha",
			repo:     Repo{Name: "alpha", Path: "/repos/alpha"},
			expected: "alpha",
		},
		{
			name:     "outside both bases uses full path",
			wtPath:   "/completely/different/path",
			repo:     Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
			expected: "/completely/different/path",
		},
		{
			name:     "empty repo path and worktree path",
			wtPath:   "/some/path",
			repo:     Repo{Name: "alpha"},
			expected: "/some/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, shortPath(tt.wtPath, tt.repo))
		})
	}
}

// ---------------------------------------------------------------------------
// buildListItems
// ---------------------------------------------------------------------------

func TestBuildListItems(t *testing.T) {
	t.Parallel()
	repos := []Repo{
		{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
		{Name: "beta", Path: "/repos/beta", WorktreePath: "/worktrees/beta"},
	}
	wts := map[string][]Worktree{
		"alpha": {
			{Repo: "alpha", Path: "/repos/alpha", Branch: "main"},
			{Repo: "alpha", Path: "/worktrees/alpha/feature", Branch: "feature"},
		},
		"beta": {
			{Repo: "beta", Path: "/repos/beta", Branch: "main"},
		},
	}

	items := buildListItems(repos, wts)
	require.Len(t, items, 3)

	// Items should be ordered: alpha items first, then beta.
	li0 := items[0].(listItem)
	assert.Equal(t, "alpha", li0.wt.Repo)
	assert.Equal(t, "main", li0.wt.Branch)

	li1 := items[1].(listItem)
	assert.Equal(t, "alpha", li1.wt.Repo)
	assert.Equal(t, "feature", li1.wt.Branch)
	assert.Equal(t, "feature", li1.displayPath)

	li2 := items[2].(listItem)
	assert.Equal(t, "beta", li2.wt.Repo)
}

func TestBuildListItems_EmptyWorktrees(t *testing.T) {
	t.Parallel()
	repos := []Repo{
		{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
	}
	wts := map[string][]Worktree{}

	items := buildListItems(repos, wts)
	assert.Empty(t, items)
}

func TestBuildListItems_MissingRepoInMap(t *testing.T) {
	t.Parallel()
	repos := []Repo{
		{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
		{Name: "gamma", Path: "/repos/gamma", WorktreePath: "/worktrees/gamma"},
	}
	wts := map[string][]Worktree{
		"alpha": {
			{Repo: "alpha", Path: "/repos/alpha", Branch: "main"},
		},
		// gamma is missing from the map.
	}

	items := buildListItems(repos, wts)
	require.Len(t, items, 1)
	assert.Equal(t, "alpha", items[0].(listItem).wt.Repo)
}

// ---------------------------------------------------------------------------
// listItem.FilterValue
// ---------------------------------------------------------------------------

func TestFilterValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		item     listItem
		expected string
	}{
		{
			name:     "normal branch",
			item:     listItem{wt: Worktree{Repo: "alpha", Branch: "main"}},
			expected: "alpha main",
		},
		{
			name:     "detached HEAD",
			item:     listItem{wt: Worktree{Repo: "beta", IsDetached: true}},
			expected: "beta (detached)",
		},
		{
			name:     "empty branch",
			item:     listItem{wt: Worktree{Repo: "gamma"}},
			expected: "gamma (unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.FilterValue())
		})
	}
}

// ---------------------------------------------------------------------------
// Delegate dimensions
// ---------------------------------------------------------------------------

func TestDelegate_HeightAndSpacing(t *testing.T) {
	t.Parallel()
	d := NewDelegate(DefaultStyles())

	assert.Equal(t, 1, d.Height())
	assert.Equal(t, 0, d.Spacing())
}
