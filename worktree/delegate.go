package worktree

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// listItem represents a single row in the grouped worktree list.
// It is either a repo header or a worktree entry. Both satisfy list.Item.
type listItem struct {
	isHeader bool
	repo     string   // set for both headers and worktree entries
	wt       Worktree // set only for worktree entries
}

// FilterValue implements list.Item. Repo headers return empty string so they
// are excluded from filter results; worktree entries filter on branch name.
func (li listItem) FilterValue() string {
	if li.isHeader {
		return ""
	}
	return li.wt.DisplayBranch()
}

// buildListItems creates a flat list of list.Items from the grouped worktree
// data, preserving repo order as provided. Each repo gets a header item
// followed by its worktree entries.
func buildListItems(repos []Repo, worktrees map[string][]Worktree) []list.Item {
	var items []list.Item
	for _, repo := range repos {
		wts, ok := worktrees[repo.Name]
		if !ok {
			continue
		}
		items = append(items, listItem{isHeader: true, repo: repo.Name})
		for _, wt := range wts {
			items = append(items, listItem{isHeader: false, repo: repo.Name, wt: wt})
		}
	}
	return items
}

// Delegate is the list.ItemDelegate for rendering worktree list items.
// It renders repo headers as bold section titles and worktree entries with
// branch, SHA, path, and status indicators.
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

// Render implements list.ItemDelegate. It writes the rendered item to w.
func (d Delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	li, ok := item.(listItem)
	if !ok {
		return
	}

	if li.isHeader {
		fmt.Fprint(w, d.Styles.RepoHeader.Render(li.repo))
		return
	}

	wt := li.wt
	isSelected := index == m.Index()

	branch := d.Styles.Branch.Render(wt.DisplayBranch())
	sha := d.Styles.SHA.Render(wt.ShortHEAD())
	path := d.Styles.Path.Render(wt.Path)

	var dirty string
	if wt.IsDirty {
		dirty = " " + d.Styles.DirtyMarker.Render("*")
	}

	var bare string
	if wt.IsBare {
		bare = " " + d.Styles.BareMarker.Render("(bare)")
	}

	if isSelected {
		branch = d.Styles.SelectedItem.Render(wt.DisplayBranch())
		line := fmt.Sprintf("> %s  %s  %s%s%s", branch, sha, path, dirty, bare)
		fmt.Fprint(w, d.Styles.SelectedItem.Render(line))
	} else {
		line := fmt.Sprintf("  %s  %s  %s%s%s", branch, sha, path, dirty, bare)
		fmt.Fprint(w, d.Styles.WorktreeItem.Render(line))
	}
}
