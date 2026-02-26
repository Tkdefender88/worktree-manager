package worktree

import "fmt"

// mockService is a configurable mock implementation of the Service interface
// for unit testing the TUI model without real git operations.
type mockService struct {
	// listFn is called by List. If nil, listResult is returned.
	listFn func(repoPath string) ([]Worktree, error)
	// listResult is a static return value for List when listFn is nil.
	listResult map[string][]Worktree

	// createFn is called by Create. If nil, createErr is returned.
	createFn  func(repoPath, worktreePath, branch string, newBranch bool) error
	createErr error

	// deleteFn is called by Delete. If nil, deleteErr is returned.
	deleteFn  func(repoPath, worktreePath string, force bool) error
	deleteErr error

	// pruneFn is called by Prune. If nil, pruneResult/pruneErr are returned.
	pruneFn     func(repoPath string, dryRun bool) ([]string, error)
	pruneResult []string
	pruneErr    error

	// Call tracking.
	listCalls   []listCall
	createCalls []createCall
	deleteCalls []deleteCall
	pruneCalls  []pruneCall
}

type listCall struct {
	RepoPath string
}

type createCall struct {
	RepoPath     string
	WorktreePath string
	Branch       string
	NewBranch    bool
}

type deleteCall struct {
	RepoPath     string
	WorktreePath string
	Force        bool
}

type pruneCall struct {
	RepoPath string
	DryRun   bool
}

func (m *mockService) List(repoPath string) ([]Worktree, error) {
	m.listCalls = append(m.listCalls, listCall{RepoPath: repoPath})
	if m.listFn != nil {
		return m.listFn(repoPath)
	}
	if wts, ok := m.listResult[repoPath]; ok {
		return wts, nil
	}
	return nil, nil
}

func (m *mockService) Create(repoPath, worktreePath, branch string, newBranch bool) error {
	m.createCalls = append(m.createCalls, createCall{
		RepoPath:     repoPath,
		WorktreePath: worktreePath,
		Branch:       branch,
		NewBranch:    newBranch,
	})
	if m.createFn != nil {
		return m.createFn(repoPath, worktreePath, branch, newBranch)
	}
	return m.createErr
}

func (m *mockService) Delete(repoPath, worktreePath string, force bool) error {
	m.deleteCalls = append(m.deleteCalls, deleteCall{
		RepoPath:     repoPath,
		WorktreePath: worktreePath,
		Force:        force,
	})
	if m.deleteFn != nil {
		return m.deleteFn(repoPath, worktreePath, force)
	}
	return m.deleteErr
}

func (m *mockService) Prune(repoPath string, dryRun bool) ([]string, error) {
	m.pruneCalls = append(m.pruneCalls, pruneCall{
		RepoPath: repoPath,
		DryRun:   dryRun,
	})
	if m.pruneFn != nil {
		return m.pruneFn(repoPath, dryRun)
	}
	return m.pruneResult, m.pruneErr
}

// --- Convenience constructors ---

// newMockService creates a mockService that returns the given worktrees for
// each repo path. All mutation operations succeed with no side effects.
func newMockService(worktrees map[string][]Worktree) *mockService {
	return &mockService{listResult: worktrees}
}

// newFailingMockService creates a mockService where List always fails.
func newFailingMockService(err error) *mockService {
	return &mockService{
		listFn: func(string) ([]Worktree, error) {
			return nil, err
		},
	}
}

// Compile-time check that mockService satisfies Service.
var _ Service = (*mockService)(nil)

// --- Test fixtures ---

func testRepos() []Repo {
	return []Repo{
		{Name: "alpha", Path: "/repos/alpha", WorktreePath: "/worktrees/alpha"},
		{Name: "beta", Path: "/repos/beta", WorktreePath: "/worktrees/beta"},
	}
}

func testWorktrees() map[string][]Worktree {
	return map[string][]Worktree{
		"alpha": {
			{Repo: "alpha", Path: "/repos/alpha", Branch: "main", HEAD: "aaaa000000000000000000000000000000000000", IsMain: true},
			{Repo: "alpha", Path: "/worktrees/alpha/feature", Branch: "feature", HEAD: "bbbb000000000000000000000000000000000000"},
		},
		"beta": {
			{Repo: "beta", Path: "/repos/beta", Branch: "main", HEAD: "cccc000000000000000000000000000000000000", IsMain: true},
		},
	}
}

// singleRepo returns a single-repo fixture for tests that need to skip the
// repo picker in create/prune flows.
func singleRepo() []Repo {
	return []Repo{
		{Name: "solo", Path: "/repos/solo", WorktreePath: "/worktrees/solo"},
	}
}

func singleRepoWorktrees() map[string][]Worktree {
	return map[string][]Worktree{
		"solo": {
			{Repo: "solo", Path: "/repos/solo", Branch: "main", HEAD: "dddd000000000000000000000000000000000000", IsMain: true},
			{Repo: "solo", Path: "/worktrees/solo/dev", Branch: "dev", HEAD: "eeee000000000000000000000000000000000000"},
		},
	}
}

// dirtyWorktree returns a Worktree fixture with IsDirty set.
func dirtyWorktree() Worktree {
	return Worktree{
		Repo:    "alpha",
		Path:    "/worktrees/alpha/dirty",
		Branch:  "dirty-branch",
		HEAD:    "ffff000000000000000000000000000000000000",
		IsDirty: true,
	}
}

// errTest is a reusable test error.
var errTest = fmt.Errorf("mock error")
