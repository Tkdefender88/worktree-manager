// Package tui implements the standalone root model for the worktree manager.
// It composes the worktree.Model component, demonstrating the intended
// embedding pattern that consumers would use in their own applications.
package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/davecgh/go-spew/spew"

	"github.com/Tkdefender88/worktree-manager/worktree"
)

// Model is the root tea.Model for the standalone worktree manager TUI.
// It wraps the worktree component and handles application-level concerns
// like debug logging and quit keybindings.
type Model struct {
	worktree worktree.Model
	debug    io.Writer
}

// Option configures the root Model.
type Option func(*Model)

// WithDebug enables debug message dumping to the given writer.
// All tea.Msg values are dumped using go-spew before being forwarded
// to the worktree component.
func WithDebug(w io.Writer) Option {
	return func(m *Model) {
		m.debug = w
	}
}

// New creates a new root Model wrapping the given worktree component.
func New(wt worktree.Model, opts ...Option) Model {
	m := Model{worktree: wt}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// Init implements tea.Model. Delegates to the worktree component.
func (m Model) Init() tea.Cmd {
	return m.worktree.Init()
}

// Update implements tea.Model. It handles application-level messages
// (quit, resize) and delegates everything to the worktree component.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.debug != nil {
		spew.Fdump(m.debug, msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.worktree.SetSize(msg.Width, msg.Height)
	}

	updated, cmd := m.worktree.Update(msg)
	m.worktree = updated.(worktree.Model)
	return m, cmd
}

// View implements tea.Model. Delegates to the worktree component.
func (m Model) View() string {
	return m.worktree.View()
}
