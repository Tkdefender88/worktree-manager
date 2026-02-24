package worktree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortHEAD(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		head     string
		expected string
	}{
		{
			name:     "full SHA",
			head:     "abc1234567890abcdef1234567890abcdef123456",
			expected: "abc1234",
		},
		{
			name:     "exactly 7 characters",
			head:     "abc1234",
			expected: "abc1234",
		},
		{
			name:     "short SHA (less than 7)",
			head:     "abc12",
			expected: "abc12",
		},
		{
			name:     "empty HEAD",
			head:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wt := Worktree{HEAD: tt.head}
			assert.Equal(t, tt.expected, wt.ShortHEAD())
		})
	}
}

func TestDisplayBranch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		wt       Worktree
		expected string
	}{
		{
			name:     "normal branch",
			wt:       Worktree{Branch: "main"},
			expected: "main",
		},
		{
			name:     "nested branch",
			wt:       Worktree{Branch: "feature/auth/login"},
			expected: "feature/auth/login",
		},
		{
			name:     "detached HEAD",
			wt:       Worktree{IsDetached: true},
			expected: "(detached)",
		},
		{
			name:     "detached HEAD with branch set (detached takes priority)",
			wt:       Worktree{Branch: "main", IsDetached: true},
			expected: "(detached)",
		},
		{
			name:     "empty branch",
			wt:       Worktree{Branch: ""},
			expected: "(unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.wt.DisplayBranch())
		})
	}
}
