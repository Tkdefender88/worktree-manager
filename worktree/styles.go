package worktree

import "github.com/charmbracelet/lipgloss"

// Styles defines the visual styles used by the worktree delegate and
// error views. List-level styles (title, help, pagination, status bar) are
// managed by list.Styles and configured separately via list.Model.Styles.
type Styles struct {
	// RepoHeader is the style for repo group headers in the list.
	RepoHeader lipgloss.Style
	// WorktreeItem is the style for a normal (unselected) worktree line.
	WorktreeItem lipgloss.Style
	// SelectedItem is the style for the currently selected worktree.
	SelectedItem lipgloss.Style
	// Branch is the style for branch names.
	Branch lipgloss.Style
	// SHA is the style for commit SHAs.
	SHA lipgloss.Style
	// Path is the style for worktree filesystem paths.
	Path lipgloss.Style
	// DirtyMarker is the style for the dirty indicator (*).
	DirtyMarker lipgloss.Style
	// BareMarker is the style for bare repo indicators.
	BareMarker lipgloss.Style
	// ErrorText is the style for error messages.
	ErrorText lipgloss.Style
}

// DefaultStyles returns the default color scheme for the delegate.
func DefaultStyles() Styles {
	return Styles{
		RepoHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			PaddingLeft(2),
		WorktreeItem: lipgloss.NewStyle().
			PaddingLeft(4),
		SelectedItem: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("170")).
			Bold(true),
		Branch: lipgloss.NewStyle().
			Foreground(lipgloss.Color("35")),
		SHA: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Path: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		DirtyMarker: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")),
		BareMarker: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true),
		ErrorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
	}
}
