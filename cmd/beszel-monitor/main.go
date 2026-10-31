package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/1325ok/Beszel-View-TUI/internal/app"
	"github.com/1325ok/Beszel-View-TUI/internal/config"
)

func main() {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "configuration:", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx, cfg, os.Stdout); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, "beszel-monitor:", err)
		os.Exit(1)
	}
}
