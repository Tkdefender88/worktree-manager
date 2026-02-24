# Worktree Manager TUI - Architecture & Implementation Plan

## Overview

A Bubble Tea TUI application for managing git worktrees across multiple repositories. Designed as both a standalone CLI binary (Cobra + Viper) and an importable library for composition into larger TUI dashboards.

Follows practices from [Tips for building Bubble Tea programs](https://leg100.github.io/en/posts/building-bubbletea-programs).

## Design Principles

1. **Keep the event loop fast** - All git operations (list, create, delete, prune) run as `tea.Cmd`s in goroutines, never blocking `Update()`
2. **Build a tree of models** - Clean separation between root model (standalone) and the worktree component model (reusable)
3. **Layout arithmetic uses lipgloss measurements** - No hardcoded heights; use `lipgloss.Height()` / `lipgloss.Width()`
4. **Debug message dumping** - Support `DEBUG` env var to dump all messages to a file
5. **Git abstraction interface** - Enables testing with mocks via `teatest`
6. **Standard tea.Model interface** - The component implements `tea.Model` for maximum compatibility with the Bubble Tea ecosystem
7. **Exported event messages** - The component emits events (created, deleted, switched, pruned) that parent models can react to

## Config File

Default location: `~/.config/worktree-manager/config.yaml`

```yaml
# Default directory where worktrees are created
# Structure: <worktree_path>/<repo-name>/<branch-name>
worktree_path: ~/worktrees

# Git repositories to manage
repos:
  - name: worktree-manager
    path: ~/workspace/work/worktree-manager
    # Optional: override the global worktree_path for this repo
    # worktree_path: ~/workspace/worktrees/worktree-manager

  - name: api-service
    path: ~/workspace/work/api-service

  - name: frontend
    path: ~/workspace/work/frontend
    worktree_path: ~/projects/frontend-trees
```

## Project Structure

```
worktree-manager/
├── main.go                          # Cobra entry point
├── go.mod                           # github.com/Tkdefender88/worktree-manager
├── go.sum
├── plan.md                          # This file
│
├── cmd/
│   └── root.go                      # Cobra root command, viper config loading
│
├── internal/
│   ├── config/
│   │   └── config.go                # Config structs + validation
│   └── git/
│       └── worktree.go              # Real git worktree.Service implementation
│
└── worktree/                        # PUBLIC - importable library
    ├── model.go                     # tea.Model (the main component)
    ├── commands.go                  # tea.Cmd functions (async git operations)
    ├── messages.go                  # Internal + exported event messages
    ├── keymap.go                    # Configurable key bindings
    ├── styles.go                    # Configurable lipgloss styles
    ├── service.go                   # Service interface definition
    ├── types.go                     # Worktree, Repo data types
    └── delegate.go                  # List item delegate rendering
```

## Configuration Flow

```
                    Standalone (Cobra/Viper)
                    ========================
    config.yaml ──> viper ──> config.Config
                                   │
                                   ▼
                            cmd/root.go builds:
                            - []worktree.Repo from config.Repos
                            - git.Service per repo
                            - worktree.New(svc, worktree.WithRepos(...), worktree.WithWorktreePath(...))
                                   │
                                   ▼
                            tea.NewProgram(rootModel{worktree: model})


                    Embedded (Dashboard)
                    ========================
                    Parent app builds config however it wants:
                            - worktree.New(svc, worktree.WithRepos(...), worktree.WithWorktreePath(...))
                            - Embeds model directly
```

## Service Interface

```go
// Service abstracts git worktree operations
type Service interface {
    List(repoPath string) ([]Worktree, error)
    Create(repoPath string, worktreePath string, branch string, newBranch bool) error
    Delete(repoPath string, worktreePath string, force bool) error
    Prune(repoPath string, dryRun bool) ([]string, error)
}
```

Each method takes `repoPath` so a single service instance handles all repos. The real implementation runs `git -C <repoPath> worktree ...`.

## Domain Types

```go
type Repo struct {
    Name         string   // display name
    Path         string   // path to the bare/main repo
    WorktreePath string   // base directory for this repo's worktrees
}

type Worktree struct {
    Repo       string    // which repo this belongs to
    Path       string    // filesystem path
    Branch     string
    HEAD       string    // commit SHA
    IsBare     bool
    IsMain     bool
    IsDirty    bool
    IsDetached bool
}
```

## Model Options

```go
func New(svc Service, opts ...Option) Model

type Option func(*Model)

func WithRepos(repos []Repo) Option
func WithDefaultWorktreePath(path string) Option
func WithKeyMap(km KeyMap) Option
func WithStyles(s Styles) Option
func WithDebug(w io.Writer) Option
```

## UI Layout

### Normal state (list view)

```
 Worktrees (3 repos, 7 trees)                       /: filter

  worktree-manager
    ● master         a1b2c3  ~/worktrees/worktree-manager/master
    ● feature-auth   d4e5f6  ~/worktrees/worktree-manager/feature-auth *

  api-service
    ● main           789abc  ~/worktrees/api-service/main
    ● fix/login      def012  ~/worktrees/api-service/fix-login

  frontend
    ● develop        345678  ~/projects/frontend-trees/develop

 n: new  d: delete  enter: switch  p: prune  ?: help  q: quit
```

### Create flow

1. User presses `n`
2. Selection: "Which repo?" → shows configured repos
3. Text input: "Branch name:"
4. Optional toggle: "Create new branch? [y/N]"
5. Worktree created at `<repo.WorktreePath>/<branch-name>`

### Delete flow

1. Cursor on a worktree, press `d`
2. Confirmation: "Delete worktree at /path/to/tree? (y/N)"
3. If dirty, warn and offer force option

### Switch flow

1. Press `enter` on a worktree
2. Emits `WorktreeSwitchedEvent{Worktree}`
3. Standalone mode: prints the path and exits

## Exported Event Messages

```go
type WorktreeCreatedEvent struct { Worktree Worktree }
type WorktreeDeletedEvent struct { Path string }
type WorktreeSwitchedEvent struct { Worktree Worktree }
type WorktreesPrunedEvent struct { Pruned []string }
```

## Implementation Phases

| Phase | Status | Description | Files |
|-------|--------|-------------|-------|
| **1. Foundation** | COMPLETE | `go mod init`, config structs, domain types, service interface | `go.mod`, `internal/config/config.go`, `worktree/types.go`, `worktree/service.go` |
| **2. Git Implementation** | COMPLETE | Real git worktree operations, parse porcelain output, unit + integration tests | `internal/git/worktree.go`, `internal/git/worktree_test.go`, `worktree/types_test.go` |
| **3. Core TUI** | COMPLETE | Model struct, `New()` with options, `Init()` loads all worktrees, list rendering grouped by repo | `worktree/model.go`, `worktree/commands.go`, `worktree/messages.go`, `worktree/delegate.go`, `worktree/styles.go`, `worktree/keymap.go` |
| **4. Operations** | | Create flow (repo picker → branch input), delete with confirmation, prune, switch | Updates to `model.go`, `commands.go`, `messages.go` |
| **5. CLI Wrapper** | | Cobra root command, viper config loading, root model wrapping the worktree model, debug dump support | `main.go`, `cmd/root.go` |
| **6. Events** | | Exported event messages for parent integration, `SetSize()` | `worktree/messages.go` |
| **7. Testing** | | Mock service, unit tests, teatest integration tests | `worktree/model_test.go`, `internal/git/worktree_test.go` |

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - List, text input, spinner, help
- `github.com/charmbracelet/lipgloss` - Styling and layout
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Config management
- `github.com/charmbracelet/x/exp/teatest` - Testing (dev)
