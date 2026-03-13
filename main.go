package main

import (
	"context"

	"github.com/Tkdefender88/worktree-manager/cmd"
)

func main() {
	ctx := context.Background()
	cmd.Execute(ctx)
}
