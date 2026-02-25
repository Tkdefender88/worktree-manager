package worktree

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// state tracks the current UI mode.
type state int

const (
	stateLoading state = iota
	stateList
	stateError

	// Create flow states.
	stateCreatePickRepo // user picks which repo to create a worktree in
	stateCreateBranch   // user types a branch name
	stateCreateConfirm  // user confirms new-branch toggle and creation
	stateCreating       // async create in progress

	// Delete flow states.
	stateDeleteConfirm // user confirms deletion (shows force option if dirty)
	stateDeleting      // async delete in progress

	// Prune flow states.
	statePrunePickRepo // user picks which repo to prune
	statePruneDryRun   // showing dry-run results, waiting for confirm
	statePruning       // async prune in progress
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
	focused   bool
	list      list.Model
	worktrees map[string][]Worktree // repo name -> worktrees
	err       error
	width     int
	height    int

	// Create flow state.
	createRepoIdx   int             // selected repo index during create
	createBranch    textinput.Model // branch name text input
	createNewBranch bool            // whether to create a new branch (-b)

	// Delete flow state.
	deleteTarget Worktree // worktree being deleted
	deleteForce  bool     // force delete (for dirty worktrees)

	// Prune flow state.
	pruneRepoIdx    int      // selected repo index during prune
	pruneDryResults []string // dry-run output to show before confirming
	pruneRepo       Repo     // repo being pruned

	// Status message shown briefly after operations.
	statusMsg string
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
		svc:     svc,
		keyMap:  DefaultKeyMap(),
		styles:  DefaultStyles(),
		state:   stateLoading,
		focused: true,
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

	// Initialize the text input for the create flow.
	ti := textinput.New()
	ti.Placeholder = "branch-name"
	ti.CharLimit = 128
	ti.Width = 40
	m.createBranch = ti

	return m
}

// SetSize updates the component dimensions. The parent model must call this
// (typically in response to tea.WindowSizeMsg) rather than forwarding
// WindowSizeMsg to Update — the component does not handle WindowSizeMsg
// itself, following the same pattern as bubbles/list and bubbles/viewport.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
}

// Focus gives the component keyboard focus. When focused, the component
// processes key messages in Update. Call this from the parent model to
// activate the component in a multi-pane layout.
func (m *Model) Focus() {
	m.focused = true
}

// Blur removes keyboard focus from the component. When blurred, the
// component ignores key messages in Update. Call this from the parent
// model to deactivate the component in a multi-pane layout.
func (m *Model) Blur() {
	m.focused = false
	// Also blur the text input if we're mid-create.
	m.createBranch.Blur()
}

// Focused reports whether the component currently has keyboard focus.
func (m Model) Focused() bool {
	return m.focused
}

// StatusMsg returns the current status message, if any. This allows parent
// models to display the status in their own chrome (e.g. a status bar)
// rather than relying on the component's built-in rendering.
func (m Model) StatusMsg() string {
	return m.statusMsg
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

// repoForWorktree finds the Repo that a worktree belongs to.
func (m Model) repoForWorktree(wt Worktree) (Repo, bool) {
	for _, r := range m.repos {
		if r.Name == wt.Repo {
			return r, true
		}
	}
	return Repo{}, false
}

// InDialog reports whether the model is currently showing a dialog
// (create, delete, prune flow) where the parent should suppress quit keys.
func (m Model) InDialog() bool {
	switch m.state {
	case stateCreatePickRepo, stateCreateBranch, stateCreateConfirm, stateCreating,
		stateDeleteConfirm, stateDeleting,
		statePrunePickRepo, statePruneDryRun, statePruning:
		return true
	}
	return false
}

// Ensure Model satisfies tea.Model at compile time.
var _ tea.Model = Model{}
