package worktree

import "fmt"

// View implements tea.Model.
func (m Model) View() string {
	switch m.state {
	case stateLoading:
		return m.viewLoading()
	case stateError:
		return m.viewError()
	case stateList:
		return m.list.View()
	default:
		return ""
	}
}

func (m Model) viewLoading() string {
	return "\n  Loading worktrees...\n"
}

func (m Model) viewError() string {
	return "\n" + m.styles.ErrorText.Render(fmt.Sprintf("  Error: %s", m.err)) + "\n"
}
