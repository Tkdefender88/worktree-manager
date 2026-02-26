package tui

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/Tkdefender88/worktree-manager/worktree"
)

// ---------------------------------------------------------------------------
// Mock service — satisfies worktree.Service for integration testing without
// real git operations.
// ---------------------------------------------------------------------------

type mockService struct {
	listFn      func(repoPath string) ([]worktree.Worktree, error)
	listResult  map[string][]worktree.Worktree
	createErr   error
	deleteErr   error
	pruneFn     func(repoPath string, dryRun bool) ([]string, error)
	pruneResult []string
	pruneErr    error
}

func (m *mockService) List(repoPath string) ([]worktree.Worktree, error) {
	if m.listFn != nil {
		return m.listFn(repoPath)
	}
	if wts, ok := m.listResult[repoPath]; ok {
		return wts, nil
	}
	return nil, nil
}

func (m *mockService) Create(repoPath, worktreePath, branch string, newBranch bool) error {
	return m.createErr
}

func (m *mockService) Delete(repoPath, worktreePath string, force bool) error {
	return m.deleteErr
}

func (m *mockService) Prune(repoPath string, dryRun bool) ([]string, error) {
	if m.pruneFn != nil {
		return m.pruneFn(repoPath, dryRun)
	}
	return m.pruneResult, m.pruneErr
}

var _ worktree.Service = (*mockService)(nil)

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

var errTest = fmt.Errorf("mock error")

func testRepos() []worktree.Repo {
	return []worktree.Repo{
		{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
		{Name: "beta", Path: "/repos/beta", WorktreePath: "/worktrees/beta"},
	}
}

func testWorktrees() map[string][]worktree.Worktree {
	return map[string][]worktree.Worktree{
		"/repos/alpha": {
			{Repo: "alpha", Path: "/repos/alpha", Branch: "main", HEAD: "aaaa000000000000000000000000000000000000", IsMain: true},
			{Repo: "alpha", Path: "/worktrees/alpha/feature", Branch: "feature", HEAD: "bbbb000000000000000000000000000000000000"},
		},
		"/repos/beta": {
			{Repo: "beta", Path: "/repos/beta", Branch: "main", HEAD: "cccc000000000000000000000000000000000000", IsMain: true},
		},
	}
}

func singleRepo() []worktree.Repo {
	return []worktree.Repo{
		{Name: "solo", Path: "/repos/solo", WorktreePath: "/worktrees/solo"},
	}
}

func singleRepoWorktrees() map[string][]worktree.Worktree {
	return map[string][]worktree.Worktree{
		"/repos/solo": {
			{Repo: "solo", Path: "/repos/solo", Branch: "main", HEAD: "dddd000000000000000000000000000000000000", IsMain: true},
			{Repo: "solo", Path: "/worktrees/solo/dev", Branch: "dev", HEAD: "eeee000000000000000000000000000000000000"},
		},
	}
}

// newApp creates a tui.Model (the real root model) backed by the given mock service.
func newApp(svc worktree.Service, repos []worktree.Repo) Model {
	wt := worktree.New(svc, worktree.WithRepos(repos))
	return New(wt)
}

// waitFor is a convenience wrapper around teatest.WaitFor that uses a single
// output reader. Each test should call tm.Output() once and pass the reader
// to all waitFor calls, since the reader is a streaming pipe.
func waitFor(t *testing.T, out io.Reader, substr string) {
	t.Helper()
	teatest.WaitFor(t, out, func(bts []byte) bool {
		return bytes.Contains(bts, []byte(substr))
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(50*time.Millisecond))
}

// ---------------------------------------------------------------------------
// Integration tests — these spin up a real tea.Program via teatest using
// the actual tui.Model root wrapper, and interact through Send/Type.
// ---------------------------------------------------------------------------

func TestIntegration_LoadAndDisplayWorktrees(t *testing.T) {
	t.Parallel()
	svc := &mockService{listResult: testWorktrees()}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	// After init, the model loads worktrees and transitions to the list view.
	waitFor(t, out, "Worktrees")

	// Quit gracefully.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_NavigateAndSwitch(t *testing.T) {
	t.Parallel()
	svc := &mockService{listResult: testWorktrees()}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Navigate down to the second item (feature worktree).
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Press enter to switch — the root model quits on WorktreeSwitchedEvent.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	final := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	m := final.(Model)
	if m.SwitchedTo() == "" {
		t.Error("expected SwitchedTo to be set after switching worktree")
	}
}

func TestIntegration_CreateFlow_SingleRepo(t *testing.T) {
	t.Parallel()
	svc := &mockService{
		listResult: singleRepoWorktrees(),
		listFn: func(repoPath string) ([]worktree.Worktree, error) {
			wts := singleRepoWorktrees()
			if w, ok := wts[repoPath]; ok {
				return w, nil
			}
			return nil, nil
		},
	}
	app := newApp(svc, singleRepo())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Press 'n' to start create — single repo skips picker, goes to branch input.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	waitFor(t, out, "Branch name")

	// Type a branch name.
	tm.Type("my-new-feature")
	waitFor(t, out, "my-new-feature")

	// Press enter to advance to confirm.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, out, "Confirm")

	// Cancel with esc to return to list.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, out, "Worktrees")

	// Quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_CreateFlow_MultiRepo(t *testing.T) {
	t.Parallel()
	svc := &mockService{
		listResult: testWorktrees(),
		listFn: func(repoPath string) ([]worktree.Worktree, error) {
			wts := testWorktrees()
			if w, ok := wts[repoPath]; ok {
				return w, nil
			}
			return nil, nil
		},
	}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Press 'n' to start create with multiple repos — should show picker.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	waitFor(t, out, "Select Repository")

	// Navigate down to beta and select it.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, out, "Branch name")

	// Type a branch name.
	tm.Type("test-branch")
	waitFor(t, out, "test-branch")

	// Press enter to go to confirm.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, out, "Confirm")

	// Esc to cancel.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, out, "Worktrees")

	// Quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_DeleteFlow(t *testing.T) {
	t.Parallel()
	wts := map[string][]worktree.Worktree{
		"/repos/alpha": {
			{Repo: "alpha", Path: "/repos/alpha", Branch: "main", HEAD: "aaaa000000000000000000000000000000000000", IsMain: true},
			{Repo: "alpha", Path: "/worktrees/alpha/feature", Branch: "feature", HEAD: "bbbb000000000000000000000000000000000000"},
		},
	}
	svc := &mockService{
		listResult: wts,
		listFn: func(repoPath string) ([]worktree.Worktree, error) {
			if w, ok := wts[repoPath]; ok {
				return w, nil
			}
			return nil, nil
		},
	}
	repos := []worktree.Repo{
		{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
	}
	app := newApp(svc, repos)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Navigate to the second item (non-main worktree).
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Press 'd' to start delete.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	waitFor(t, out, "Delete Worktree")

	// Cancel with 'n'.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	waitFor(t, out, "Worktrees")

	// Quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_PruneFlow(t *testing.T) {
	t.Parallel()
	svc := &mockService{
		listResult: singleRepoWorktrees(),
		listFn: func(repoPath string) ([]worktree.Worktree, error) {
			wts := singleRepoWorktrees()
			if w, ok := wts[repoPath]; ok {
				return w, nil
			}
			return nil, nil
		},
		pruneFn: func(repoPath string, dryRun bool) ([]string, error) {
			return []string{"/stale/worktree-1"}, nil
		},
	}
	app := newApp(svc, singleRepo())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Press 'p' to start prune — single repo skips picker.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	// Should show dry-run results.
	waitFor(t, out, "stale/worktree-1")

	// Cancel with esc.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, out, "Worktrees")

	// Quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_QuitFromList(t *testing.T) {
	t.Parallel()
	svc := &mockService{listResult: testWorktrees()}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Press 'q' to quit from list state.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_QuitBlockedInDialog(t *testing.T) {
	t.Parallel()
	svc := &mockService{listResult: testWorktrees()}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Enter create flow (dialog).
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	waitFor(t, out, "Select Repository")

	// 'q' should NOT quit while in dialog.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Navigate down in the picker — this triggers a re-render, proving we
	// are still in the dialog and didn't quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, out, "beta")

	// Esc to return to list, then quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, out, "Worktrees")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_ErrorState(t *testing.T) {
	t.Parallel()
	svc := &mockService{
		listFn: func(string) ([]worktree.Worktree, error) {
			return nil, errTest
		},
	}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	// The model should show an error after init fails.
	// The error view renders "Error: mock error" inside ANSI styling,
	// so just look for the error text itself.
	waitFor(t, out, "mock error")

	// Quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestIntegration_FinalModel_ReflectsState(t *testing.T) {
	t.Parallel()
	svc := &mockService{listResult: testWorktrees()}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	final := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	m := final.(Model)

	// FinalOutput should be empty since no worktree was switched.
	if m.FinalOutput() != "" {
		t.Errorf("expected empty FinalOutput, got %q", m.FinalOutput())
	}
}

func TestIntegration_SwitchSetsFinalOutput(t *testing.T) {
	t.Parallel()
	svc := &mockService{listResult: testWorktrees()}
	app := newApp(svc, testRepos())

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	out := tm.Output()

	waitFor(t, out, "Worktrees")

	// Press enter to switch the first worktree.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	final := tm.FinalModel(t, teatest.WithFinalTimeout(3*time.Second))
	m := final.(Model)

	if m.FinalOutput() == "" {
		t.Error("expected FinalOutput to contain the switched worktree path")
	}
	if m.SwitchedTo() == "" {
		t.Error("expected SwitchedTo to be set")
	}
}
