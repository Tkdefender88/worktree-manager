package worktree

// Internal messages used within the component.

// worktreesLoadedMsg is sent when all worktrees have been loaded from all repos.
type worktreesLoadedMsg struct {
	// worktrees maps repo name to its worktrees.
	worktrees map[string][]Worktree
}

// errMsg is sent when any async command fails.
type errMsg struct {
	err error
}

// Exported event messages that parent models can react to.

// WorktreeCreatedEvent is emitted after a worktree is successfully created.
type WorktreeCreatedEvent struct {
	Worktree Worktree
}

// WorktreeDeletedEvent is emitted after a worktree is successfully deleted.
type WorktreeDeletedEvent struct {
	Path string
}

// WorktreeSwitchedEvent is emitted when the user selects a worktree to switch to.
type WorktreeSwitchedEvent struct {
	Worktree Worktree
}

// WorktreesPrunedEvent is emitted after worktrees have been pruned.
type WorktreesPrunedEvent struct {
	Pruned []string
}
