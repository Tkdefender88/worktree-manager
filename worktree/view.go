package worktree

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const dialogPadding = 2 // left/right padding inside dialogs

// View implements tea.Model.
func (m Model) View() string {
	switch m.state {
	case stateLoading:
		return m.viewLoading()
	case stateError:
		return m.viewError()
	case stateList:
		return m.viewList()
	case stateCreatePickRepo:
		return m.viewCreatePickRepo()
	case stateCreateBranch:
		return m.viewCreateBranch()
	case stateCreateConfirm:
		return m.viewCreateConfirm()
	case stateCreating:
		return m.viewStatus()
	case stateDeleteConfirm:
		return m.viewDeleteConfirm()
	case stateDeleting:
		return m.viewStatus()
	case statePrunePickRepo:
		return m.viewPrunePickRepo()
	case statePruneDryRun:
		return m.viewPruneDryRun()
	case statePruning:
		return m.viewStatus()
	default:
		return ""
	}
}

// dialogStyle returns a lipgloss style that constrains dialog content to the
// component's current width. This ensures dialogs render correctly when the
// component is embedded at a fixed size in a parent layout.
func (m Model) dialogStyle() lipgloss.Style {
	w := m.width
	if w <= 0 {
		w = 80 // sensible default before first SetSize call
	}
	return lipgloss.NewStyle().
		Padding(1, dialogPadding).
		MaxWidth(w)
}

// contentWidth returns the usable width inside a dialog after accounting for
// padding on both sides.
func (m Model) contentWidth() int {
	w := m.width
	if w <= 0 {
		w = 80
	}
	return max(w-dialogPadding*2, 20)
}

func (m Model) viewLoading() string {
	return m.dialogStyle().Render("Loading worktrees...")
}

func (m Model) viewError() string {
	return m.dialogStyle().Render(
		m.styles.ErrorText.Render(fmt.Sprintf("Error: %s", m.err)),
	)
}

func (m Model) viewList() string {
	var b strings.Builder
	b.WriteString(m.list.View())
	if m.statusMsg != "" {
		b.WriteString("\n  " + m.styles.StatusMsg().Render(m.statusMsg))
	}
	return b.String()
}

func (m Model) viewStatus() string {
	return m.dialogStyle().Render(m.statusMsg)
}

// --- Create flow views ---

func (m Model) viewCreatePickRepo() string {
	title := m.styles.DialogTitle().Render("Create Worktree - Select Repository")

	var b strings.Builder
	b.WriteString(title + "\n\n")

	for i, repo := range m.repos {
		cursor := "  "
		name := repo.Name
		if i == m.createRepoIdx {
			cursor = m.styles.SelectedItem.Render("> ")
			name = m.styles.SelectedItem.Render(repo.Name)
		} else {
			name = m.styles.RepoName.Render(repo.Name)
		}
		b.WriteString(cursor + name + "\n")
	}

	b.WriteString("\n" + m.styles.HintText().Render("j/k: navigate  enter: select  esc: cancel"))
	return m.dialogStyle().Render(b.String())
}

func (m Model) viewCreateBranch() string {
	repo := m.repos[m.createRepoIdx]
	title := m.styles.DialogTitle().Render("Create Worktree")

	var b strings.Builder
	b.WriteString(title + "\n\n")
	b.WriteString("Repository: " + m.styles.RepoName.Render(repo.Name) + "\n")
	b.WriteString("Branch name: " + m.createBranch.View() + "\n")
	b.WriteString("\n" + m.styles.HintText().Render("enter: next  esc: cancel"))
	return m.dialogStyle().Render(b.String())
}

func (m Model) viewCreateConfirm() string {
	repo := m.repos[m.createRepoIdx]
	branch := m.createBranch.Value()
	title := m.styles.DialogTitle().Render("Create Worktree - Confirm")

	newBranchLabel := "No"
	if m.createNewBranch {
		newBranchLabel = "Yes"
	}

	var b strings.Builder
	b.WriteString(title + "\n\n")
	b.WriteString("Repository:  " + m.styles.RepoName.Render(repo.Name) + "\n")
	b.WriteString("Branch:      " + m.styles.Branch.Render(branch) + "\n")
	b.WriteString("New branch:  " + newBranchLabel + "\n")
	b.WriteString("Path:        " + m.styles.Path.Render(repo.WorktreePath+"/"+branch) + "\n")
	b.WriteString("\n" + m.styles.HintText().Render("y/enter: create  tab: toggle new branch  esc/n: cancel"))
	return m.dialogStyle().Render(b.String())
}

// --- Delete flow views ---

func (m Model) viewDeleteConfirm() string {
	wt := m.deleteTarget
	title := m.styles.DialogTitle().Render("Delete Worktree")

	var b strings.Builder
	b.WriteString(title + "\n\n")
	b.WriteString("Repository: " + m.styles.RepoName.Render(wt.Repo) + "\n")
	b.WriteString("Branch:     " + m.styles.Branch.Render(wt.DisplayBranch()) + "\n")
	b.WriteString("Path:       " + m.styles.Path.Render(wt.Path) + "\n")

	if wt.IsDirty {
		b.WriteString("\n" + m.styles.ErrorText.Render("WARNING: This worktree has uncommitted changes!") + "\n")
		if m.deleteForce {
			b.WriteString(m.styles.ErrorText.Render("Force delete is ON - changes will be lost.") + "\n")
		}
	}

	forceHint := ""
	if wt.IsDirty {
		if m.deleteForce {
			forceHint = "  f: disable force"
		} else {
			forceHint = "  f: enable force (required for dirty worktrees)"
		}
	}

	b.WriteString("\n" + m.styles.HintText().Render("y/enter: delete"+forceHint+"  esc/n: cancel"))
	return m.dialogStyle().Render(b.String())
}

// --- Prune flow views ---

func (m Model) viewPrunePickRepo() string {
	title := m.styles.DialogTitle().Render("Prune Stale Worktrees - Select Repository")

	var b strings.Builder
	b.WriteString(title + "\n\n")

	for i, repo := range m.repos {
		cursor := "  "
		name := repo.Name
		if i == m.pruneRepoIdx {
			cursor = m.styles.SelectedItem.Render("> ")
			name = m.styles.SelectedItem.Render(repo.Name)
		} else {
			name = m.styles.RepoName.Render(repo.Name)
		}
		b.WriteString(cursor + name + "\n")
	}

	b.WriteString("\n" + m.styles.HintText().Render("j/k: navigate  enter: select  esc: cancel"))
	return m.dialogStyle().Render(b.String())
}

func (m Model) viewPruneDryRun() string {
	title := m.styles.DialogTitle().Render("Prune Stale Worktrees - Confirm")

	var b strings.Builder
	b.WriteString(title + "\n\n")
	b.WriteString("Repository: " + m.styles.RepoName.Render(m.pruneRepo.Name) + "\n\n")

	if len(m.pruneDryResults) == 0 {
		b.WriteString("No stale worktrees found.\n")
	} else {
		b.WriteString("The following stale entries will be removed:\n\n")
		for _, entry := range m.pruneDryResults {
			b.WriteString("  " + m.styles.Path.Render(entry) + "\n")
		}
	}

	b.WriteString("\n" + m.styles.HintText().Render("y/enter: prune  esc/n: cancel"))
	return m.dialogStyle().Render(b.String())
}

// --- Style helpers ---

// Styles helper methods for dialog-specific styles that don't need to be in
// the exported Styles struct since they're composed from existing styles.

// DialogTitle returns a style for dialog titles.
func (s Styles) DialogTitle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Underline(true)
}

// HintText returns a style for hint/help text in dialogs.
func (s Styles) HintText() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
}

// StatusMsg returns a style for status messages.
func (s Styles) StatusMsg() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("35")).
		Italic(true)
}
