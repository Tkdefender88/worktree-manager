// Package worktree provides a Bubble Tea model for managing git worktrees
// across multiple repositories. It can be used standalone or embedded in a
// larger Bubble Tea application.
package worktree

// Repo represents a configured git repository whose worktrees are managed.
type Repo struct {
	// Name is the display name for this repository.
	Name string
	// Path is the filesystem path to the main/bare repository.
	Path string
	// WorktreePath is the base directory where worktrees for this repo are created.
	// Structure: <WorktreePath>/<branch-name>
	WorktreePath string
}

// Worktree represents a single git worktree.
type Worktree struct {
	// Repo is the name of the repository this worktree belongs to.
	Repo string
	// Path is the filesystem path of the worktree.
	Path string
	// Branch is the checked-out branch name.
	Branch string
	// HEAD is the commit SHA at the tip of the worktree.
	HEAD string
	// IsBare indicates this is the bare repository entry.
	IsBare bool
	// IsMain indicates this is the main worktree (not a linked worktree).
	IsMain bool
	// IsDirty indicates the worktree has uncommitted changes.
	IsDirty bool
	// IsDetached indicates HEAD is detached (not on a branch).
	IsDetached bool
}

// ShortHEAD returns the first 7 characters of the HEAD SHA.
func (w Worktree) ShortHEAD() string {
	if len(w.HEAD) >= 7 {
		return w.HEAD[:7]
	}
	return w.HEAD
}

// DisplayBranch returns the branch name for display, or "(detached)" if detached.
func (w Worktree) DisplayBranch() string {
	if w.IsDetached {
		return "(detached)"
	}
	if w.Branch == "" {
		return "(unknown)"
	}
	return w.Branch
}
