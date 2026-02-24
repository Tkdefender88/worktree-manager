package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Tkdefender88/worktree-manager/worktree"
)

// ---------------------------------------------------------------------------
// parsePorcelain unit tests (pure function, no git required)
// ---------------------------------------------------------------------------

func TestParsePorcelain_SingleWorktree(t *testing.T) {
	t.Parallel()
	input := []byte(strings.Join([]string{
		"worktree /home/user/repo",
		"HEAD abc1234567890abcdef1234567890abcdef123456",
		"branch refs/heads/main",
		"",
	}, "\n"))

	wts, err := parsePorcelain(input)
	require.NoError(t, err)
	require.Len(t, wts, 1)

	wt := wts[0]
	assert.Equal(t, "/home/user/repo", wt.Path)
	assert.Equal(t, "abc1234567890abcdef1234567890abcdef123456", wt.HEAD)
	assert.Equal(t, "main", wt.Branch)
	assert.True(t, wt.IsMain)
	assert.False(t, wt.IsBare)
	assert.False(t, wt.IsDetached)
}

func TestParsePorcelain_MultipleWorktrees(t *testing.T) {
	t.Parallel()
	input := []byte(strings.Join([]string{
		"worktree /home/user/repo",
		"HEAD aaaa000000000000000000000000000000000000",
		"branch refs/heads/main",
		"",
		"worktree /home/user/worktrees/feature",
		"HEAD bbbb000000000000000000000000000000000000",
		"branch refs/heads/feature-auth",
		"",
		"worktree /home/user/worktrees/fix",
		"HEAD cccc000000000000000000000000000000000000",
		"branch refs/heads/fix/login",
		"",
	}, "\n"))

	wts, err := parsePorcelain(input)
	require.NoError(t, err)
	require.Len(t, wts, 3)

	// First is always main.
	assert.True(t, wts[0].IsMain)
	assert.False(t, wts[1].IsMain)
	assert.False(t, wts[2].IsMain)

	assert.Equal(t, "feature-auth", wts[1].Branch)
	assert.Equal(t, "fix/login", wts[2].Branch)
}

func TestParsePorcelain_BareRepo(t *testing.T) {
	t.Parallel()
	input := []byte(strings.Join([]string{
		"worktree /home/user/repo.git",
		"HEAD aaaa000000000000000000000000000000000000",
		"bare",
		"",
		"worktree /home/user/worktrees/main",
		"HEAD aaaa000000000000000000000000000000000000",
		"branch refs/heads/main",
		"",
	}, "\n"))

	wts, err := parsePorcelain(input)
	require.NoError(t, err)
	require.Len(t, wts, 2)

	assert.True(t, wts[0].IsBare)
	assert.True(t, wts[0].IsMain)
	assert.Empty(t, wts[0].Branch)

	assert.False(t, wts[1].IsBare)
	assert.Equal(t, "main", wts[1].Branch)
}

func TestParsePorcelain_DetachedHEAD(t *testing.T) {
	t.Parallel()
	input := []byte(strings.Join([]string{
		"worktree /home/user/repo",
		"HEAD aaaa000000000000000000000000000000000000",
		"branch refs/heads/main",
		"",
		"worktree /home/user/worktrees/detached",
		"HEAD bbbb000000000000000000000000000000000000",
		"detached",
		"",
	}, "\n"))

	wts, err := parsePorcelain(input)
	require.NoError(t, err)
	require.Len(t, wts, 2)

	assert.True(t, wts[1].IsDetached)
	assert.Empty(t, wts[1].Branch)
}

func TestParsePorcelain_NoTrailingNewline(t *testing.T) {
	t.Parallel()
	// Some git versions don't output a trailing blank line after the last entry.
	input := []byte(strings.Join([]string{
		"worktree /home/user/repo",
		"HEAD aaaa000000000000000000000000000000000000",
		"branch refs/heads/main",
	}, "\n"))

	wts, err := parsePorcelain(input)
	require.NoError(t, err)
	require.Len(t, wts, 1)

	assert.Equal(t, "/home/user/repo", wts[0].Path)
	assert.Equal(t, "main", wts[0].Branch)
}

func TestParsePorcelain_EmptyInput(t *testing.T) {
	t.Parallel()
	wts, err := parsePorcelain([]byte(""))
	require.NoError(t, err)
	assert.Empty(t, wts)
}

func TestParsePorcelain_StripsBranchPrefix(t *testing.T) {
	t.Parallel()
	// Ensure refs/heads/ is stripped and nested branch names are preserved.
	input := []byte(strings.Join([]string{
		"worktree /repo",
		"HEAD aaaa000000000000000000000000000000000000",
		"branch refs/heads/feature/nested/deep",
		"",
	}, "\n"))

	wts, err := parsePorcelain(input)
	require.NoError(t, err)
	assert.Equal(t, "feature/nested/deep", wts[0].Branch)
}

// ---------------------------------------------------------------------------
// Integration tests using real git repos in temp directories
// ---------------------------------------------------------------------------

// initTestRepo creates a git repo in a temp dir with an initial commit.
// Returns the repo path (symlink-resolved for macOS /private/var).
func initTestRepo(t *testing.T) string {
	t.Helper()

	rawDir := t.TempDir()
	// On macOS, t.TempDir() returns /var/folders/... but git resolves symlinks
	// to /private/var/folders/..., causing path mismatches. Resolve upfront.
	dir, err := filepath.EvalSymlinks(rawDir)
	require.NoError(t, err, "resolving symlinks for temp dir")

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test User"},
		// Disable commit signing so tests work without GPG/SSH keys.
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "config", "tag.gpgsign", "false"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "setup %v: %s", args, out)
	}

	// Create an initial commit so we have a branch to work with.
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# test\n"), 0644))

	cmds = [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "setup %v: %s", args, out)
	}

	return dir
}

// resolvedTempPath creates a temp dir and returns a symlink-resolved subpath.
// This prevents macOS /var vs /private/var mismatches when comparing paths
// that git reports.
func resolvedTempPath(t *testing.T, subpath string) string {
	t.Helper()
	raw := t.TempDir()
	resolved, err := filepath.EvalSymlinks(raw)
	require.NoError(t, err, "resolving symlinks for temp dir")
	return filepath.Join(resolved, subpath)
}

func TestIsDirty_CleanRepo(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)

	dirty, err := isDirty(repo)
	require.NoError(t, err)
	assert.False(t, dirty, "clean repo should not be dirty")
}

func TestIsDirty_UncommittedChanges(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)

	// Create an untracked file.
	require.NoError(t, os.WriteFile(filepath.Join(repo, "new.txt"), []byte("hello\n"), 0644))

	dirty, err := isDirty(repo)
	require.NoError(t, err)
	assert.True(t, dirty, "repo with untracked file should be dirty")
}

func TestIsDirty_StagedChanges(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)

	// Stage a modification.
	readme := filepath.Join(repo, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("# modified\n"), 0644))

	cmd := exec.Command("git", "add", "README.md")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git add: %s", out)

	dirty, err := isDirty(repo)
	require.NoError(t, err)
	assert.True(t, dirty, "repo with staged changes should be dirty")
}

func TestIsDirty_InvalidPath(t *testing.T) {
	t.Parallel()
	_, err := isDirty("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// WorktreeService integration tests
// ---------------------------------------------------------------------------

func TestWorktreeService_List(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	wts, err := svc.List(repo)
	require.NoError(t, err)

	// A fresh repo should have exactly one worktree (the main one).
	require.Len(t, wts, 1)

	wt := wts[0]
	assert.Equal(t, repo, wt.Path)
	assert.True(t, wt.IsMain)
	assert.False(t, wt.IsBare)
	assert.False(t, wt.IsDetached)
	assert.False(t, wt.IsDirty)
	assert.NotEmpty(t, wt.Branch)
	assert.NotEmpty(t, wt.HEAD)
}

func TestWorktreeService_CreateAndList_ExistingBranch(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	// Create a branch to check out in the worktree.
	cmd := exec.Command("git", "branch", "feature-test")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "creating branch: %s", out)

	wtPath := resolvedTempPath(t, "feature-test")

	require.NoError(t, svc.Create(repo, wtPath, "feature-test", false))

	// Verify the worktree directory was created.
	assert.DirExists(t, wtPath)

	// List should now show 2 worktrees.
	wts, err := svc.List(repo)
	require.NoError(t, err)
	require.Len(t, wts, 2)

	// Find the new worktree.
	found := findWorktreeByPath(wts, wtPath)
	require.NotNil(t, found, "new worktree not found in list")

	assert.Equal(t, "feature-test", found.Branch)
	assert.False(t, found.IsMain)
}

func TestWorktreeService_CreateAndList_NewBranch(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	wtPath := resolvedTempPath(t, "new-branch")

	require.NoError(t, svc.Create(repo, wtPath, "new-branch", true))

	// Verify the worktree and branch were created.
	wts, err := svc.List(repo)
	require.NoError(t, err)
	require.Len(t, wts, 2)

	found := findWorktreeByPath(wts, wtPath)
	require.NotNil(t, found, "new worktree not found in list")

	assert.Equal(t, "new-branch", found.Branch)
}

func TestWorktreeService_Delete(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	wtPath := resolvedTempPath(t, "to-delete")

	// Create a worktree to delete.
	require.NoError(t, svc.Create(repo, wtPath, "delete-me", true))

	// Confirm it exists.
	wts, err := svc.List(repo)
	require.NoError(t, err)
	require.Len(t, wts, 2)

	// Delete it.
	require.NoError(t, svc.Delete(repo, wtPath, false))

	// Verify it's gone from the list.
	wts, err = svc.List(repo)
	require.NoError(t, err)
	assert.Len(t, wts, 1)

	// Verify directory is removed.
	assert.NoDirExists(t, wtPath)
}

func TestWorktreeService_Delete_Force(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	wtPath := resolvedTempPath(t, "force-delete")

	require.NoError(t, svc.Create(repo, wtPath, "force-branch", true))

	// Make the worktree dirty.
	require.NoError(t, os.WriteFile(filepath.Join(wtPath, "dirty.txt"), []byte("dirty\n"), 0644))

	// Non-force delete should fail on a dirty worktree.
	assert.Error(t, svc.Delete(repo, wtPath, false), "expected Delete() without force to fail on dirty worktree")

	// Force delete should succeed.
	require.NoError(t, svc.Delete(repo, wtPath, true))

	wts, err := svc.List(repo)
	require.NoError(t, err)
	assert.Len(t, wts, 1)
}

func TestWorktreeService_Prune_DryRun(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	// On a clean repo, prune should return nothing.
	pruned, err := svc.Prune(repo, true)
	require.NoError(t, err)
	assert.Empty(t, pruned)
}

func TestWorktreeService_Prune_StaleWorktree(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	wtPath := resolvedTempPath(t, "stale")

	// Create a worktree, then manually remove its directory to make it stale.
	require.NoError(t, svc.Create(repo, wtPath, "stale-branch", true))

	// Manually remove the worktree directory (simulates stale reference).
	require.NoError(t, os.RemoveAll(wtPath))

	// Dry run should report the stale entry.
	pruned, err := svc.Prune(repo, true)
	require.NoError(t, err)
	assert.NotEmpty(t, pruned, "expected at least 1 stale worktree to report")

	// List should still show 2 worktrees (stale one not yet pruned).
	wts, err := svc.List(repo)
	require.NoError(t, err)
	assert.Len(t, wts, 2)

	// Real prune should clean it up.
	pruned, err = svc.Prune(repo, false)
	require.NoError(t, err)
	assert.NotEmpty(t, pruned, "expected at least 1 stale worktree to be pruned")

	// Now list should be back to 1.
	wts, err = svc.List(repo)
	require.NoError(t, err)
	assert.Len(t, wts, 1)
}

func TestWorktreeService_List_DirtyDetection(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()

	wtPath := resolvedTempPath(t, "dirty-wt")

	require.NoError(t, svc.Create(repo, wtPath, "dirty-branch", true))

	// Make the worktree dirty.
	require.NoError(t, os.WriteFile(filepath.Join(wtPath, "untracked.txt"), []byte("data\n"), 0644))

	wts, err := svc.List(repo)
	require.NoError(t, err)

	dirtyWT := findWorktreeByPath(wts, wtPath)
	require.NotNil(t, dirtyWT, "dirty worktree not found in list")
	assert.True(t, dirtyWT.IsDirty)

	// The main worktree should be clean.
	mainWT := findMainWorktree(wts)
	require.NotNil(t, mainWT, "main worktree not found in list")
	assert.False(t, mainWT.IsDirty)
}

func TestWorktreeService_Create_InvalidRepo(t *testing.T) {
	t.Parallel()
	svc := NewWorktreeService()
	assert.Error(t, svc.Create("/nonexistent/path", "/tmp/wt", "branch", true))
}

func TestWorktreeService_List_InvalidRepo(t *testing.T) {
	t.Parallel()
	svc := NewWorktreeService()
	_, err := svc.List("/nonexistent/path")
	assert.Error(t, err)
}

func TestWorktreeService_Delete_InvalidPath(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)
	svc := NewWorktreeService()
	assert.Error(t, svc.Delete(repo, "/nonexistent/worktree", false))
}

// ---------------------------------------------------------------------------
// runGit tests
// ---------------------------------------------------------------------------

func TestRunGit_Success(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)

	out, err := runGit(repo, "status", "--porcelain")
	require.NoError(t, err)
	// Clean repo should have empty porcelain output.
	assert.Empty(t, strings.TrimSpace(string(out)))
}

func TestRunGit_InvalidCommand(t *testing.T) {
	t.Parallel()
	repo := initTestRepo(t)

	_, err := runGit(repo, "not-a-real-command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not-a-real-command")
}

func TestRunGit_InvalidDir(t *testing.T) {
	t.Parallel()
	_, err := runGit("/nonexistent/dir", "status")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func findWorktreeByPath(wts []worktree.Worktree, path string) *worktree.Worktree {
	for i := range wts {
		if wts[i].Path == path {
			return &wts[i]
		}
	}
	return nil
}

func findMainWorktree(wts []worktree.Worktree) *worktree.Worktree {
	for i := range wts {
		if wts[i].IsMain {
			return &wts[i]
		}
	}
	return nil
}
