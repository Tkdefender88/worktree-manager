package worktree

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the application-specific key bindings for the worktree
// component. Navigation, filtering, and help keys are managed by the
// underlying list.Model and do not need to be duplicated here.
type KeyMap struct {
	Create key.Binding
	Delete key.Binding
	Switch key.Binding
	Prune  key.Binding
}

// DefaultKeyMap returns the default application-specific key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Create: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Switch: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "switch"),
		),
		Prune: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prune"),
		),
	}
}

// ShortHelp returns bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Create, k.Delete, k.Switch, k.Prune}
}

// FullHelp returns bindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Create, k.Delete},
		{k.Switch, k.Prune},
	}
}
