package worktree

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// listItem represents a single worktree entry in the list. Every item is
// selectable — there are no header-only rows.
type listItem struct {
	wt          Worktree
	displayPath string // path relative to the repo's configured worktree base
}

// FilterValue implements list.Item. Returns the repo name and branch so
// users can filter by either.
func (li listItem) FilterValue() string {
	return li.wt.Repo + " " + li.wt.DisplayBranch()
}

// buildListItems creates a flat list of list.Items from the grouped worktree
// data, preserving repo order as provided. Each item's displayPath is computed
// relative to the repo's configured WorktreePath.
func buildListItems(repos []Repo, worktrees map[string][]Worktree) []list.Item {
	var items []list.Item
	for _, repo := range repos {
		wts, ok := worktrees[repo.Name]
		if !ok {
			continue
		}
		for _, wt := range wts {
			display := shortPath(wt.Path, repo)
			items = append(items, listItem{wt: wt, displayPath: display})
		}
	}
	return items
}

// shortPath returns a display-friendly path for the worktree. It tries to
// make the path relative to the repo's WorktreePath first (for linked
// worktrees), then relative to the repo's own Path (for the main checkout).
// Falls back to the full path if neither produces a clean relative path.
func shortPath(wtPath string, repo Repo) string {
	// Try the configured worktree base directory first.
	if repo.WorktreePath != "" {
		if rel, err := filepath.Rel(repo.WorktreePath, wtPath); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	// Try relative to the repo's parent directory (covers the main checkout).
	if repo.Path != "" {
		parent := filepath.Dir(repo.Path)
		if rel, err := filepath.Rel(parent, wtPath); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return wtPath
}

// Delegate is the list.ItemDelegate for rendering worktree list items.
// Each item is rendered as a single line with the repo name as a colored
// prefix for visual grouping.
type Delegate struct {
	Styles Styles
}

// NewDelegate creates a Delegate with the given styles.
func NewDelegate(styles Styles) Delegate {
	return Delegate{Styles: styles}
}

// Height implements list.ItemDelegate.
func (d Delegate) Height() int { return 1 }

// Spacing implements list.ItemDelegate.
func (d Delegate) Spacing() int { return 0 }

// Update implements list.ItemDelegate.
func (d Delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

const maxBranchLen = 12

// truncate shortens s to maxLen runes, appending an ellipsis if truncated.
// The first maxLen characters are always visible.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

// branchColumn formats a branch name into a fixed-width column.
// Truncated names show maxBranchLen chars + ellipsis (maxBranchLen+1 display cols).
// Short names are padded to maxBranchLen+1 display cols to keep columns aligned.
func branchColumn(name string, maxLen int) string {
	truncated := truncate(name, maxLen)
	runes := []rune(truncated)
	width := maxLen + 1 // account for ellipsis column
	if len(runes) < width {
		return truncated + strings.Repeat(" ", width-len(runes))
	}
	return truncated
}

// Render implements list.ItemDelegate. It writes the rendered item to w.
// Format: [cursor] repo-name  branch  sha  path [*] [(bare)]
func (d Delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	li, ok := item.(listItem)
	if !ok {
		return
	}

	wt := li.wt
	isSelected := index == m.Index()

	branchText := branchColumn(wt.DisplayBranch(), maxBranchLen)

	repo := d.Styles.RepoName.Render(wt.Repo)
	branch := d.Styles.Branch.Render(branchText)
	sha := d.Styles.SHA.Render(wt.ShortHEAD())
	path := d.Styles.Path.Render(li.displayPath)

	var dirty string
	if wt.IsDirty {
		dirty = " " + d.Styles.DirtyMarker.Render("*")
	}

	var bare string
	if wt.IsBare {
		bare = " " + d.Styles.BareMarker.Render("(bare)")
	}

	cursor := "  "
	if isSelected {
		cursor = d.Styles.SelectedItem.Render("> ")
		branch = d.Styles.SelectedItem.Render(branchText)
	}

	fmt.Fprintf(w, "%s%s  %s  %s  %s%s%s", cursor, repo, branch, sha, path, dirty, bare)
}
