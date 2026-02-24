package worktree

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// state tracks the current UI mode.
type state int

const (
	stateLoading state = iota
	stateList
	stateError
)

// Model is the Bubble Tea model for the worktree manager component.
// It implements tea.Model and can be used standalone or embedded in a
// larger Bubble Tea application.
//
// Internally it wraps a bubbles/list.Model for navigation, filtering,
// pagination, and help rendering.
type Model struct {
	// Configuration.
	svc              Service
	repos            []Repo
	defaultWorktrees string // default base directory for worktrees
	keyMap           KeyMap
	styles           Styles

	// UI state.
	state     state
	list      list.Model
	worktrees map[string][]Worktree // repo name -> worktrees
	err       error
	width     int
	height    int
}

// Option configures the Model.
type Option func(*Model)

// WithRepos sets the repos to manage.
func WithRepos(repos []Repo) Option {
	return func(m *Model) {
		m.repos = repos
	}
}

// WithDefaultWorktreePath sets the default base directory for new worktrees.
func WithDefaultWorktreePath(path string) Option {
	return func(m *Model) {
		m.defaultWorktrees = path
	}
}

// WithKeyMap overrides the default application-specific key bindings.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) {
		m.keyMap = km
	}
}

// WithStyles overrides the default delegate styles.
func WithStyles(s Styles) Option {
	return func(m *Model) {
		m.styles = s
	}
}

// New creates a new worktree Model with the given service and options.
func New(svc Service, opts ...Option) Model {
	m := Model{
		svc:    svc,
		keyMap: DefaultKeyMap(),
		styles: DefaultStyles(),
		state:  stateLoading,
	}
	for _, opt := range opts {
		opt(&m)
	}

	// Initialize the list with an empty set of items and our custom delegate.
	// Items will be populated asynchronously via Init().
	delegate := NewDelegate(m.styles)
	m.list = list.New(nil, delegate, 0, 0)
	m.list.Title = "Worktrees"
	m.list.SetShowStatusBar(true)
	m.list.SetStatusBarItemName("worktree", "worktrees")
	m.list.SetFilteringEnabled(true)
	m.list.DisableQuitKeybindings()

	// Inject our app-specific key bindings into the list's help view.
	km := m.keyMap
	m.list.AdditionalShortHelpKeys = km.ShortHelp
	m.list.AdditionalFullHelpKeys = func() []key.Binding {
		var bindings []key.Binding
		for _, group := range km.FullHelp() {
			bindings = append(bindings, group...)
		}
		return bindings
	}

	return m
}

// SetSize updates the component dimensions. Call this from the parent model
// when the terminal is resized.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
}

// SelectedWorktree returns the currently selected worktree, if any.
func (m Model) SelectedWorktree() (Worktree, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return Worktree{}, false
	}
	li, ok := item.(listItem)
	if !ok {
		return Worktree{}, false
	}
	return li.wt, true
}

// Ensure Model satisfies tea.Model at compile time.
var _ tea.Model = Model{}
