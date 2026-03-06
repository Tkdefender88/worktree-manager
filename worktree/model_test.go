package worktree

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper: run a sequence of messages through Update and return the final model.
// Any tea.Cmd returned is executed synchronously (one level deep) and fed back
// as a message if it produces one. This is intentionally simple — it handles
// the common case of sync-returning cmds without a full tea.Program.
// ---------------------------------------------------------------------------

// sendMsg sends a single message through Update and returns the updated model.
// It does NOT execute the returned cmd — use sendMsgAndCollect for that.
func sendMsg(t *testing.T, m tea.Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// sendMsgAndCmd sends a message and returns both the model and the cmd.
func sendMsgAndCmd(t *testing.T, m tea.Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

// execCmd runs a tea.Cmd and returns the resulting message, or nil if cmd is nil.
func execCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	return cmd()
}

// execBatch runs a tea.BatchMsg and collects all non-nil results.
func execBatch(t *testing.T, msg tea.Msg) []tea.Msg {
	t.Helper()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var results []tea.Msg
	for _, cmd := range batch {
		if result := execCmd(t, cmd); result != nil {
			results = append(results, result)
		}
	}
	return results
}

// keyMsg creates a tea.KeyMsg from a string (e.g. "enter", "n", "esc").
func keyMsg(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// specialKeyMsg creates a tea.KeyMsg for special keys.
func specialKeyMsg(k tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: k}
}

// newTestModel creates a model pre-loaded with worktrees (past the loading state).
func newTestModel(t *testing.T, svc *mockService, repos []Repo, wts map[string][]Worktree) Model {
	t.Helper()
	m := New(svc, WithRepos(repos))
	m.SetSize(80, 24)
	// Simulate the worktrees being loaded.
	m = sendMsg(t, m, worktreesLoadedMsg{worktrees: wts})
	require.Equal(t, stateList, m.state)
	return m
}

// ---------------------------------------------------------------------------
// New() and Option tests
// ---------------------------------------------------------------------------

func TestNew_DefaultState(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)

	assert.Equal(t, stateLoading, m.state)
	assert.True(t, m.focused, "model should be focused by default")
	assert.Empty(t, m.repos)
	assert.Equal(t, "Worktrees", m.list.Title)
}

func TestNew_WithRepos(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	repos := testRepos()
	m := New(svc, WithRepos(repos))

	assert.Equal(t, repos, m.repos)
}

func TestNew_WithDefaultWorktreePath(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc, WithDefaultWorktreePath("/custom/path"))

	assert.Equal(t, "/custom/path", m.defaultWorktrees)
}

func TestNew_WithKeyMap(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	km := DefaultKeyMap()
	m := New(svc, WithKeyMap(km))

	assert.Equal(t, km, m.keyMap)
}

func TestNew_WithStyles(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	s := DefaultStyles()
	m := New(svc, WithStyles(s))

	// Just verify it was set (styles are value types, so equality works).
	assert.Equal(t, s, m.styles)
}

// ---------------------------------------------------------------------------
// SetSize, Focus, Blur
// ---------------------------------------------------------------------------

func TestSetSize(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)

	m.SetSize(120, 40)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestFocusBlur(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)

	assert.True(t, m.Focused())

	m.Blur()
	assert.False(t, m.Focused())

	m.Focus()
	assert.True(t, m.Focused())
}

// ---------------------------------------------------------------------------
// InDialog
// ---------------------------------------------------------------------------

func TestInDialog(t *testing.T) {
	t.Parallel()

	dialogStates := []state{
		stateCreatePickRepo, stateCreateBranch, stateCreateConfirm, stateCreating,
		stateDeleteConfirm, stateDeleting,
		statePrunePickRepo, statePruneDryRun, statePruning,
	}
	nonDialogStates := []state{
		stateLoading, stateList, stateError,
	}

	for _, s := range dialogStates {
		m := Model{state: s}
		assert.True(t, m.InDialog(), "state %d should be a dialog state", s)
	}
	for _, s := range nonDialogStates {
		m := Model{state: s}
		assert.False(t, m.InDialog(), "state %d should not be a dialog state", s)
	}
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestInit_ReturnsLoadCmd(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := New(svc, WithRepos(repos))

	cmd := m.Init()
	require.NotNil(t, cmd, "Init should return a non-nil cmd")

	// Init returns a batch (loadWorktrees + spinner.Tick). Execute the batch
	// and look for the worktreesLoadedMsg among the results.
	msg := execCmd(t, cmd)
	results := execBatch(t, msg)

	var loaded worktreesLoadedMsg
	var found bool
	for _, r := range results {
		if wl, ok := r.(worktreesLoadedMsg); ok {
			loaded = wl
			found = true
		}
	}
	require.True(t, found, "Init batch should contain worktreesLoadedMsg")
	assert.Len(t, loaded.worktrees, 2)
	assert.Len(t, svc.listCalls, 2)
}

func TestInit_ServiceError(t *testing.T) {
	t.Parallel()
	svc := newFailingMockService(errTest)
	repos := testRepos()
	m := New(svc, WithRepos(repos))

	cmd := m.Init()
	msg := execCmd(t, cmd)
	results := execBatch(t, msg)

	var found bool
	for _, r := range results {
		if errResult, ok := r.(errMsg); ok {
			assert.Equal(t, errTest, errResult.err)
			found = true
		}
	}
	require.True(t, found, "expected errMsg in Init batch results")
}

// ---------------------------------------------------------------------------
// Update: worktreesLoadedMsg
// ---------------------------------------------------------------------------

func TestUpdate_WorktreesLoaded(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	repos := testRepos()
	m := New(svc, WithRepos(repos))
	m.SetSize(80, 24)

	wts := testWorktrees()
	m = sendMsg(t, m, worktreesLoadedMsg{worktrees: wts})

	assert.Equal(t, stateList, m.state)
	assert.Equal(t, wts, m.worktrees)
	// The list should contain items for all worktrees (2 from alpha + 1 from beta).
	assert.Equal(t, 3, len(m.list.Items()))
}

// ---------------------------------------------------------------------------
// Update: errMsg
// ---------------------------------------------------------------------------

func TestUpdate_ErrMsg(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)

	m = sendMsg(t, m, errMsg{err: errTest})

	assert.Equal(t, stateError, m.state)
	assert.Equal(t, errTest, m.err)
}

// ---------------------------------------------------------------------------
// Blurred model ignores key messages
// ---------------------------------------------------------------------------

func TestUpdate_BlurredIgnoresKeys(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m.Blur()
	// Pressing 'n' (create) should be ignored.
	updated := sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateList, updated.state, "blurred model should stay in list state")
}

func TestUpdate_BlurredProcessesNonKeyMsgs(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)
	m.Blur()

	// Non-key message (errMsg) should still be processed.
	updated := sendMsg(t, m, errMsg{err: errTest})
	assert.Equal(t, stateError, updated.state, "blurred model should process non-key msgs")
}

// ---------------------------------------------------------------------------
// Switch flow
// ---------------------------------------------------------------------------

func TestSwitch_EmitsEvent(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	// Press enter to switch.
	updated, cmd := sendMsgAndCmd(t, m, specialKeyMsg(tea.KeyEnter))
	_ = updated
	require.NotNil(t, cmd)

	msg := execCmd(t, cmd)
	evt, ok := msg.(WorktreeSwitchedEvent)
	require.True(t, ok, "expected WorktreeSwitchedEvent, got %T", msg)
	// The first item in the list should be the first worktree.
	assert.NotEmpty(t, evt.Worktree.Branch)
}

// ---------------------------------------------------------------------------
// Create flow — multi-repo path
// ---------------------------------------------------------------------------

func TestCreate_MultiRepo_PickRepoThenBranchThenConfirm(t *testing.T) {
	t.Parallel()
	repos := testRepos()
	svc := &mockService{
		listResult: testWorktrees(),
	}
	m := newTestModel(t, svc, repos, testWorktrees())

	// Press 'n' to start create flow.
	m = sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateCreatePickRepo, m.state)
	assert.Equal(t, 0, m.createRepoIdx)

	// Navigate down to pick second repo.
	m = sendMsg(t, m, keyMsg("j"))
	assert.Equal(t, 1, m.createRepoIdx)

	// Navigate back up.
	m = sendMsg(t, m, keyMsg("k"))
	assert.Equal(t, 0, m.createRepoIdx)

	// Can't go above 0.
	m = sendMsg(t, m, keyMsg("k"))
	assert.Equal(t, 0, m.createRepoIdx)

	// Select the first repo.
	m = sendMsg(t, m, specialKeyMsg(tea.KeyEnter))
	assert.Equal(t, stateCreateBranch, m.state)

	// Type a branch name via text input rune messages.
	for _, r := range "my-feature" {
		m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	assert.Equal(t, "my-feature", m.createBranch.Value())

	// Press enter to move to confirm.
	m = sendMsg(t, m, specialKeyMsg(tea.KeyEnter))
	assert.Equal(t, stateCreateConfirm, m.state)
	assert.True(t, m.createNewBranch, "new branch should default to true")

	// Toggle new branch off.
	m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyTab})
	assert.False(t, m.createNewBranch)

	// Toggle back on.
	m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyTab})
	assert.True(t, m.createNewBranch)

	// Confirm creation.
	m, cmd := sendMsgAndCmd(t, m, keyMsg("y"))
	assert.Equal(t, stateCreating, m.state)
	assert.Equal(t, "Creating worktree...", m.statusMsg)
	require.NotNil(t, cmd)
}

func TestCreate_MultiRepo_EscCancels(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m = sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateCreatePickRepo, m.state)

	m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, stateList, m.state)
}

func TestCreate_BranchEsc_ReturnsList(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	// Enter create flow.
	m = sendMsg(t, m, keyMsg("n"))
	m = sendMsg(t, m, specialKeyMsg(tea.KeyEnter))
	assert.Equal(t, stateCreateBranch, m.state)

	// Esc out.
	m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, stateList, m.state)
}

func TestCreate_EmptyBranch_StaysInBranchState(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m = sendMsg(t, m, keyMsg("n"))
	m = sendMsg(t, m, specialKeyMsg(tea.KeyEnter))
	assert.Equal(t, stateCreateBranch, m.state)

	// Press enter with empty branch name — should stay.
	m = sendMsg(t, m, specialKeyMsg(tea.KeyEnter))
	assert.Equal(t, stateCreateBranch, m.state)
}

func TestCreate_ConfirmEsc_ReturnsList(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	// Navigate to confirm state.
	m = sendMsg(t, m, keyMsg("n"))
	m = sendMsg(t, m, specialKeyMsg(tea.KeyEnter))
	for _, r := range "branch" {
		m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m = sendMsg(t, m, specialKeyMsg(tea.KeyEnter))
	assert.Equal(t, stateCreateConfirm, m.state)

	// Cancel with 'n'.
	m = sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateList, m.state)
}

func TestCreate_NoRepos_NothingHappens(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := newTestModel(t, svc, nil, nil)

	m = sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateList, m.state, "create with no repos should be a no-op")
}

// ---------------------------------------------------------------------------
// Create flow — single repo path (skips picker)
// ---------------------------------------------------------------------------

func TestCreate_SingleRepo_SkipsPicker(t *testing.T) {
	t.Parallel()
	repos := singleRepo()
	svc := newMockService(singleRepoWorktrees())
	m := newTestModel(t, svc, repos, singleRepoWorktrees())

	m = sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateCreateBranch, m.state, "single repo should skip picker")
	assert.Equal(t, 0, m.createRepoIdx)
}

// ---------------------------------------------------------------------------
// Create flow — worktreeCreatedMsg handling
// ---------------------------------------------------------------------------

func TestUpdate_WorktreeCreatedMsg(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	wt := Worktree{Repo: "alpha", Branch: "new-branch", Path: "/worktrees/alpha/new-branch"}
	m, cmd := sendMsgAndCmd(t, m, worktreeCreatedMsg{wt: wt})

	assert.Equal(t, stateLoading, m.state)
	assert.Contains(t, m.statusMsg, "new-branch")
	require.NotNil(t, cmd)

	// The cmd should be a batch: reload + spinner tick + emit event.
	msg := execCmd(t, cmd)
	results := execBatch(t, msg)
	require.NotEmpty(t, results)

	// One of the results should be a WorktreeCreatedEvent.
	var found bool
	for _, r := range results {
		if evt, ok := r.(WorktreeCreatedEvent); ok {
			assert.Equal(t, "new-branch", evt.Worktree.Branch)
			found = true
		}
	}
	assert.True(t, found, "expected WorktreeCreatedEvent in batch results")
}

// ---------------------------------------------------------------------------
// Delete flow
// ---------------------------------------------------------------------------

func TestDelete_ConfirmAndExecute(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	// Navigate to a non-main worktree (second item = alpha/feature).
	m = sendMsg(t, m, keyMsg("j"))

	// Start delete.
	m = sendMsg(t, m, keyMsg("d"))
	assert.Equal(t, stateDeleteConfirm, m.state)
	assert.Equal(t, "feature", m.deleteTarget.Branch)
	assert.False(t, m.deleteForce)

	// Confirm deletion.
	m, cmd := sendMsgAndCmd(t, m, keyMsg("y"))
	assert.Equal(t, stateDeleting, m.state)
	assert.Equal(t, "Deleting worktree...", m.statusMsg)
	require.NotNil(t, cmd)

	// The cmd is a batch (deleteWorktree + spinner.Tick). Execute and find
	// the worktreeDeletedMsg among the results.
	msg := execCmd(t, cmd)
	results := execBatch(t, msg)

	var found bool
	for _, r := range results {
		if deleted, ok := r.(worktreeDeletedMsg); ok {
			assert.Equal(t, "/worktrees/alpha/feature", deleted.path)
			found = true
		}
	}
	require.True(t, found, "expected worktreeDeletedMsg in batch results")
}

func TestDelete_CancelWithEsc(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("d"))
	assert.Equal(t, stateDeleteConfirm, m.state)

	m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, stateList, m.state)
}

func TestDelete_CancelWithN(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("d"))
	assert.Equal(t, stateDeleteConfirm, m.state)

	m = sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateList, m.state)
}

func TestDelete_MainWorktree_Blocked(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	// First item is the main worktree (alpha/main).
	m = sendMsg(t, m, keyMsg("d"))
	assert.Equal(t, stateList, m.state, "deleting main worktree should be blocked")
	assert.Contains(t, m.statusMsg, "Cannot delete")
}

func TestDelete_BareWorktree_Blocked(t *testing.T) {
	t.Parallel()
	wts := map[string][]Worktree{
		"alpha": {
			{Repo: "alpha", Path: "/repos/alpha.git", IsBare: true, IsMain: true, HEAD: "aaaa000000000000000000000000000000000000"},
		},
	}
	svc := newMockService(wts)
	repos := []Repo{{Name: "alpha", Path: "/repos/alpha.git", WorktreePath: "/worktrees/alpha"}}
	m := newTestModel(t, svc, repos, wts)

	m = sendMsg(t, m, keyMsg("d"))
	assert.Equal(t, stateList, m.state, "deleting bare worktree should be blocked")
	assert.Contains(t, m.statusMsg, "Cannot delete")
}

func TestDelete_ForceToggle(t *testing.T) {
	t.Parallel()
	dirty := dirtyWorktree()
	wts := map[string][]Worktree{
		"alpha": {
			{Repo: "alpha", Path: "/repos/alpha", Branch: "main", HEAD: "aaaa000000000000000000000000000000000000", IsMain: true},
			dirty,
		},
	}
	svc := newMockService(wts)
	repos := testRepos()
	m := newTestModel(t, svc, repos, wts)

	// Select the dirty worktree (second item).
	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("d"))
	assert.Equal(t, stateDeleteConfirm, m.state)
	assert.False(t, m.deleteForce)

	// Toggle force.
	m = sendMsg(t, m, keyMsg("f"))
	assert.True(t, m.deleteForce)

	// Toggle force off.
	m = sendMsg(t, m, keyMsg("f"))
	assert.False(t, m.deleteForce)
}

// ---------------------------------------------------------------------------
// Delete flow — worktreeDeletedMsg handling
// ---------------------------------------------------------------------------

func TestUpdate_WorktreeDeletedMsg(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m, cmd := sendMsgAndCmd(t, m, worktreeDeletedMsg{path: "/worktrees/alpha/feature"})
	assert.Equal(t, stateLoading, m.state)
	assert.Equal(t, "Deleted worktree", m.statusMsg)
	require.NotNil(t, cmd)

	msg := execCmd(t, cmd)
	results := execBatch(t, msg)

	var found bool
	for _, r := range results {
		if evt, ok := r.(WorktreeDeletedEvent); ok {
			assert.Equal(t, "/worktrees/alpha/feature", evt.Path)
			found = true
		}
	}
	assert.True(t, found, "expected WorktreeDeletedEvent in batch results")
}

// ---------------------------------------------------------------------------
// Prune flow — multi-repo path
// ---------------------------------------------------------------------------

func TestPrune_MultiRepo_PickRepoThenDryRunThenConfirm(t *testing.T) {
	t.Parallel()
	svc := &mockService{
		listResult: testWorktrees(),
		pruneFn: func(repoPath string, dryRun bool) ([]string, error) {
			if dryRun {
				return []string{"/stale/entry1", "/stale/entry2"}, nil
			}
			return []string{"/stale/entry1", "/stale/entry2"}, nil
		},
	}
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	// Press 'p' to start prune.
	m = sendMsg(t, m, keyMsg("p"))
	assert.Equal(t, statePrunePickRepo, m.state)

	// Navigate down.
	m = sendMsg(t, m, keyMsg("j"))
	assert.Equal(t, 1, m.pruneRepoIdx)

	// Navigate back up.
	m = sendMsg(t, m, keyMsg("k"))
	assert.Equal(t, 0, m.pruneRepoIdx)

	// Can't go below 0.
	m = sendMsg(t, m, keyMsg("k"))
	assert.Equal(t, 0, m.pruneRepoIdx)

	// Select repo.
	m, cmd := sendMsgAndCmd(t, m, specialKeyMsg(tea.KeyEnter))
	assert.Equal(t, statePruning, m.state)
	require.NotNil(t, cmd)

	// The cmd is a batch (pruneWorktreesDryRun + spinner.Tick). Execute and
	// find the pruneDryRunMsg among the results.
	msg := execCmd(t, cmd)
	results := execBatch(t, msg)

	var dryResult pruneDryRunMsg
	var ok bool
	for _, r := range results {
		if dr, isDry := r.(pruneDryRunMsg); isDry {
			dryResult = dr
			ok = true
		}
	}
	require.True(t, ok, "expected pruneDryRunMsg in batch results")
	assert.Len(t, dryResult.pruned, 2)

	// Feed the dry-run result back.
	m = sendMsg(t, m, dryResult)
	assert.Equal(t, statePruneDryRun, m.state)
	assert.Len(t, m.pruneDryResults, 2)

	// Confirm prune.
	m, cmd = sendMsgAndCmd(t, m, keyMsg("y"))
	assert.Equal(t, statePruning, m.state)
	assert.Equal(t, "Pruning...", m.statusMsg)
	require.NotNil(t, cmd)
}

func TestPrune_MultiRepo_EscCancels(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m = sendMsg(t, m, keyMsg("p"))
	assert.Equal(t, statePrunePickRepo, m.state)

	m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, stateList, m.state)
}

func TestPrune_DryRun_NothingToPrune(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())
	m.state = statePruning // simulate being in pruning state

	m = sendMsg(t, m, pruneDryRunMsg{repo: repos[0], pruned: nil})
	assert.Equal(t, stateList, m.state)
	assert.Equal(t, "Nothing to prune", m.statusMsg)
}

func TestPrune_DryRun_EscCancels(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())
	m.state = statePruneDryRun
	m.pruneDryResults = []string{"/stale"}

	m = sendMsg(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, stateList, m.state)
}

func TestPrune_NoRepos_NothingHappens(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := newTestModel(t, svc, nil, nil)

	m = sendMsg(t, m, keyMsg("p"))
	assert.Equal(t, stateList, m.state, "prune with no repos should be a no-op")
}

// ---------------------------------------------------------------------------
// Prune flow — single repo path (skips picker)
// ---------------------------------------------------------------------------

func TestPrune_SingleRepo_SkipsPicker(t *testing.T) {
	t.Parallel()
	repos := singleRepo()
	svc := &mockService{
		listResult:  singleRepoWorktrees(),
		pruneResult: []string{},
	}
	m := newTestModel(t, svc, repos, singleRepoWorktrees())

	m, cmd := sendMsgAndCmd(t, m, keyMsg("p"))
	assert.Equal(t, statePruning, m.state, "single repo should skip picker")
	require.NotNil(t, cmd)
}

// ---------------------------------------------------------------------------
// Prune flow — worktreesPrunedMsg handling
// ---------------------------------------------------------------------------

func TestUpdate_WorktreesPrunedMsg(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m, cmd := sendMsgAndCmd(t, m, worktreesPrunedMsg{pruned: []string{"/stale/1"}})
	assert.Equal(t, stateLoading, m.state)
	assert.Equal(t, "Pruned stale worktrees", m.statusMsg)
	require.NotNil(t, cmd)

	msg := execCmd(t, cmd)
	results := execBatch(t, msg)

	var found bool
	for _, r := range results {
		if evt, ok := r.(WorktreesPrunedEvent); ok {
			assert.Equal(t, []string{"/stale/1"}, evt.Pruned)
			found = true
		}
	}
	assert.True(t, found, "expected WorktreesPrunedEvent in batch results")
}

// ---------------------------------------------------------------------------
// SelectedWorktree
// ---------------------------------------------------------------------------

func TestSelectedWorktree_ReturnsSelected(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	wt, ok := m.SelectedWorktree()
	assert.True(t, ok)
	assert.NotEmpty(t, wt.Branch)
}

func TestSelectedWorktree_EmptyList(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := newTestModel(t, svc, testRepos(), map[string][]Worktree{})

	wt, ok := m.SelectedWorktree()
	assert.False(t, ok)
	assert.Equal(t, Worktree{}, wt)
}

// ---------------------------------------------------------------------------
// StatusMsg
// ---------------------------------------------------------------------------

func TestStatusMsg_Empty(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)

	assert.Empty(t, m.StatusMsg())
}

func TestStatusMsg_Set(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)
	m.statusMsg = "test status"

	assert.Equal(t, "test status", m.StatusMsg())
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_LoadingState(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)

	view := m.View()
	assert.Contains(t, view, "Loading worktrees...")
}

func TestView_ErrorState(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)
	m = sendMsg(t, m, errMsg{err: fmt.Errorf("something broke")})

	view := m.View()
	assert.Contains(t, view, "something broke")
}

func TestView_ListState(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	view := m.View()
	assert.NotEmpty(t, view)
}

func TestView_CreatePickRepoState(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())
	m = sendMsg(t, m, keyMsg("n"))

	view := m.View()
	assert.Contains(t, view, "Select Repository")
	assert.Contains(t, view, "alpha")
	assert.Contains(t, view, "beta")
}

func TestView_DeleteConfirmState(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	// Select non-main worktree and start delete.
	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("d"))

	view := m.View()
	assert.Contains(t, view, "Delete Worktree")
	assert.Contains(t, view, "feature")
}

func TestView_DeleteConfirmDirtyWarning(t *testing.T) {
	t.Parallel()
	dirty := dirtyWorktree()
	wts := map[string][]Worktree{
		"alpha": {
			{Repo: "alpha", Path: "/repos/alpha", Branch: "main", HEAD: "aaaa000000000000000000000000000000000000", IsMain: true},
			dirty,
		},
	}
	svc := newMockService(wts)
	repos := testRepos()
	m := newTestModel(t, svc, repos, wts)

	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("d"))

	view := m.View()
	assert.Contains(t, view, "uncommitted changes")
}

func TestView_PruneDryRunState(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())
	m.state = statePruneDryRun
	m.pruneRepo = repos[0]
	m.pruneDryResults = []string{"/stale/entry1", "/stale/entry2"}

	view := m.View()
	assert.Contains(t, view, "Confirm")
	assert.Contains(t, view, "/stale/entry1")
	assert.Contains(t, view, "/stale/entry2")
}

func TestView_StatusState(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	m := New(svc)
	m.state = stateCreating
	m.statusMsg = "Creating worktree..."

	view := m.View()
	assert.Contains(t, view, "Creating worktree...")
}

// ---------------------------------------------------------------------------
// repoForWorktree
// ---------------------------------------------------------------------------

func TestRepoForWorktree_Found(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	repos := testRepos()
	m := New(svc, WithRepos(repos))

	repo, ok := m.repoForWorktree(Worktree{Repo: "alpha"})
	assert.True(t, ok)
	assert.Equal(t, "alpha", repo.Name)
}

func TestRepoForWorktree_NotFound(t *testing.T) {
	t.Parallel()
	svc := newMockService(nil)
	repos := testRepos()
	m := New(svc, WithRepos(repos))

	_, ok := m.repoForWorktree(Worktree{Repo: "nonexistent"})
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Commands unit tests (run the cmd functions directly with mock service)
// ---------------------------------------------------------------------------

func TestLoadWorktrees_Success(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()

	cmd := loadWorktrees(svc, repos)
	msg := execCmd(t, cmd)

	loaded, ok := msg.(worktreesLoadedMsg)
	require.True(t, ok)
	assert.Len(t, loaded.worktrees, 2)

	// Verify repo names are tagged.
	for _, wt := range loaded.worktrees["alpha"] {
		assert.Equal(t, "alpha", wt.Repo)
	}
}

func TestLoadWorktrees_Error(t *testing.T) {
	t.Parallel()
	svc := newFailingMockService(errTest)
	repos := testRepos()

	cmd := loadWorktrees(svc, repos)
	msg := execCmd(t, cmd)

	errResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Equal(t, errTest, errResult.err)
}

func TestCreateWorktree_Success(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"}
	created := Worktree{
		Path:   "/worktrees/alpha/my-branch",
		Branch: "my-branch",
		HEAD:   "1234567890abcdef",
	}
	svc := &mockService{
		listFn: func(repoPath string) ([]Worktree, error) {
			return []Worktree{created}, nil
		},
	}

	cmd := createWorktree(svc, repo, "my-branch", true)
	msg := execCmd(t, cmd)

	result, ok := msg.(worktreeCreatedMsg)
	require.True(t, ok)
	assert.Equal(t, "my-branch", result.wt.Branch)
	assert.Equal(t, "alpha", result.wt.Repo)

	// Verify create was called.
	require.Len(t, svc.createCalls, 1)
	assert.Equal(t, "/repos/alpha", svc.createCalls[0].RepoPath)
	assert.Equal(t, "/worktrees/alpha/my-branch", svc.createCalls[0].WorktreePath)
	assert.True(t, svc.createCalls[0].NewBranch)
}

func TestCreateWorktree_CreateFails(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"}
	svc := &mockService{
		createErr: errTest,
	}

	cmd := createWorktree(svc, repo, "branch", true)
	msg := execCmd(t, cmd)

	errResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Equal(t, errTest, errResult.err)
}

func TestCreateWorktree_ListFailsAfterCreate(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"}
	callCount := 0
	svc := &mockService{
		listFn: func(repoPath string) ([]Worktree, error) {
			callCount++
			return nil, errTest
		},
	}

	cmd := createWorktree(svc, repo, "branch", false)
	msg := execCmd(t, cmd)

	// Should still return worktreeCreatedMsg with partial data.
	result, ok := msg.(worktreeCreatedMsg)
	require.True(t, ok, "expected worktreeCreatedMsg even when list fails, got %T", msg)
	assert.Equal(t, "branch", result.wt.Branch)
	assert.Equal(t, "alpha", result.wt.Repo)
}

func TestDeleteWorktree_Success(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"}
	wt := Worktree{Path: "/worktrees/alpha/feature"}
	svc := &mockService{}

	cmd := deleteWorktree(svc, repo, wt, false)
	msg := execCmd(t, cmd)

	result, ok := msg.(worktreeDeletedMsg)
	require.True(t, ok)
	assert.Equal(t, "/worktrees/alpha/feature", result.path)

	require.Len(t, svc.deleteCalls, 1)
	assert.False(t, svc.deleteCalls[0].Force)
}

func TestDeleteWorktree_Error(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"}
	wt := Worktree{Path: "/worktrees/alpha/feature"}
	svc := &mockService{deleteErr: errTest}

	cmd := deleteWorktree(svc, repo, wt, false)
	msg := execCmd(t, cmd)

	errResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Equal(t, errTest, errResult.err)
}

func TestPruneWorktreesDryRun_Success(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha"}
	svc := &mockService{
		pruneResult: []string{"/stale/1", "/stale/2"},
	}

	cmd := pruneWorktreesDryRun(svc, repo)
	msg := execCmd(t, cmd)

	result, ok := msg.(pruneDryRunMsg)
	require.True(t, ok)
	assert.Len(t, result.pruned, 2)
	assert.Equal(t, repo, result.repo)

	require.Len(t, svc.pruneCalls, 1)
	assert.True(t, svc.pruneCalls[0].DryRun)
}

func TestPruneWorktreesDryRun_Error(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha"}
	svc := &mockService{pruneErr: errTest}

	cmd := pruneWorktreesDryRun(svc, repo)
	msg := execCmd(t, cmd)

	errResult, ok := msg.(errMsg)
	require.True(t, ok)
	assert.Equal(t, errTest, errResult.err)
}

func TestPruneWorktrees_Success(t *testing.T) {
	t.Parallel()
	repo := Repo{Name: "alpha", Path: "/repos/alpha"}
	svc := &mockService{
		pruneResult: []string{"/pruned/1"},
	}

	cmd := pruneWorktrees(svc, repo)
	msg := execCmd(t, cmd)

	result, ok := msg.(worktreesPrunedMsg)
	require.True(t, ok)
	assert.Equal(t, []string{"/pruned/1"}, result.pruned)

	require.Len(t, svc.pruneCalls, 1)
	assert.False(t, svc.pruneCalls[0].DryRun)
}

// ---------------------------------------------------------------------------
// Prune flow — navigate beyond bounds
// ---------------------------------------------------------------------------

func TestPrune_PickRepo_NavigateBeyondMax(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m = sendMsg(t, m, keyMsg("p"))
	assert.Equal(t, statePrunePickRepo, m.state)

	// Navigate past the end.
	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("j"))
	assert.Equal(t, len(repos)-1, m.pruneRepoIdx, "should not exceed max index")
}

func TestCreate_PickRepo_NavigateBeyondMax(t *testing.T) {
	t.Parallel()
	svc := newMockService(testWorktrees())
	repos := testRepos()
	m := newTestModel(t, svc, repos, testWorktrees())

	m = sendMsg(t, m, keyMsg("n"))
	assert.Equal(t, stateCreatePickRepo, m.state)

	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("j"))
	m = sendMsg(t, m, keyMsg("j"))
	assert.Equal(t, len(repos)-1, m.createRepoIdx, "should not exceed max index")
}
