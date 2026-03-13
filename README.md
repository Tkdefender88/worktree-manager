# Worktree Manager

A terminal UI for managing git worktrees across multiple repositories. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Designed as both a standalone CLI and an importable `tea.Model` component for embedding in larger TUI applications.

## Features

- List worktrees grouped by repository
- Create new worktrees (with optional new branch)
- Delete worktrees (with force option for dirty trees)
- Prune stale worktree references (dry-run + confirm)
- Switch to a worktree (prints path on exit for `cd` integration)
- Filter worktrees by name or branch
- Configurable key bindings and styles

## Install

```sh
go install github.com/Tkdefender88/worktree-manager@latest
```

## Configuration

Create a config file at `~/.config/worktree-manager/config.yaml`:

```yaml
# Default directory where worktrees are created
worktree_path: ~/worktrees

# Git repositories to manage
repos:
  - name: my-project
    path: ~/workspace/my-project

  - name: api-service
    path: ~/workspace/api-service

  - name: frontend
    path: ~/workspace/frontend
    worktree_path: ~/projects/frontend-trees  # per-repo override
```

## Usage

### Adding a repository

From inside any directory in a git repository, run:

```sh
worktree-manager init
```

You will be prompted for a project name. If the config file does not exist yet you will also be prompted for the default worktree base path.

Pass `-n`/`--name` to skip the prompt:

```sh
worktree-manager init -n my-project
```

This finds the nearest `.git` root (searching upward from the current directory), adds it to the config, and creates the config file if it does not already exist.

### Launching the TUI

```sh
worktree-manager
```

### Key bindings

| Key     | Action                         |
|---------|--------------------------------|
| `j`/`k` | Navigate up/down              |
| `enter` | Switch to selected worktree   |
| `n`     | Create a new worktree         |
| `d`     | Delete selected worktree      |
| `p`     | Prune stale worktrees         |
| `/`     | Filter worktrees              |
| `q`     | Quit                          |

### Shell integration

Switch to a worktree and `cd` into it:

```sh
cd "$(worktree-manager)"
```

Or add a shell function:

```sh
wt() { cd "$(worktree-manager)" || return; }
```

### Flags

Global flags (available on all subcommands):

```
--config string   config file (default ~/.config/worktree-manager/config.yaml)
--debug           dump all tea.Msg values to debug.log
```

`init` flags:

```
-n, --name string   project name to use in the config
```

## Library usage

The `worktree` package exports a `tea.Model` component that can be embedded in any Bubble Tea application:

```go
import "github.com/Tkdefender88/worktree-manager/worktree"

svc := git.NewWorktreeService()
model := worktree.New(svc,
    worktree.WithRepos(repos),
    worktree.WithDefaultWorktreePath("~/worktrees"),
)
```

The component emits events that parent models can handle:

- `WorktreeCreatedEvent` -- a worktree was created
- `WorktreeDeletedEvent` -- a worktree was deleted
- `WorktreeSwitchedEvent` -- the user selected a worktree
- `WorktreesPrunedEvent` -- stale worktrees were pruned

The parent model is responsible for calling `SetSize(width, height)` on resize and `Focus()`/`Blur()` for multi-pane layouts.

## Project structure

```
worktree-manager/
├── main.go                     # Entry point
├── cmd/
│   ├── root.go                 # Cobra CLI, Viper config loading
│   └── init.go                 # `init` subcommand
├── internal/
│   ├── config/config.go        # Config structs + validation
│   ├── git/worktree.go         # Real git worktree operations
│   └── tui/model.go            # Root tea.Model (standalone wrapper)
└── worktree/                   # Public library package
    ├── model.go                # tea.Model component
    ├── update.go               # Update logic and state machine
    ├── view.go                 # View rendering
    ├── commands.go             # Async tea.Cmd functions
    ├── messages.go             # Internal + exported event messages
    ├── service.go              # Service interface
    ├── types.go                # Worktree, Repo data types
    ├── delegate.go             # List item rendering
    ├── keymap.go               # Key bindings
    └── styles.go               # Lipgloss styles
```

## Development

```sh
# Run tests
go test ./...

# Run tests with race detector
go test ./... -race

# Build
go build -o worktree-manager .
```

Tests include unit tests with a mock `Service` implementation and integration tests using [teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest) that run a real `tea.Program` and interact with it through simulated key presses.
