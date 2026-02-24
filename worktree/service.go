package worktree

// Service abstracts git worktree operations for testability.
// Each method takes a repoPath so a single Service instance can handle
// multiple repositories.
type Service interface {
	// List returns all worktrees for the repository at repoPath.
	List(repoPath string) ([]Worktree, error)

	// Create creates a new worktree. If newBranch is true, the branch is
	// created from HEAD before checking it out in the worktree.
	Create(repoPath string, worktreePath string, branch string, newBranch bool) error

	// Delete removes a worktree. If force is true, the worktree is removed
	// even if it has uncommitted changes.
	Delete(repoPath string, worktreePath string, force bool) error

	// Prune removes stale worktree references. If dryRun is true, it reports
	// what would be pruned without actually removing anything.
	Prune(repoPath string, dryRun bool) ([]string, error)
}
