package app

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/1325ok/Beszel-View-TUI/internal/api"
	"github.com/1325ok/Beszel-View-TUI/internal/config"
	"github.com/1325ok/Beszel-View-TUI/internal/render"
	"github.com/1325ok/Beszel-View-TUI/internal/screen"
	"github.com/1325ok/Beszel-View-TUI/internal/state"
	"github.com/1325ok/Beszel-View-TUI/internal/terminal"
)

func Run(ctx context.Context, cfg config.Config, out io.Writer) error {
	restoreTerminal, err := terminal.Prepare(out)
	if err != nil {
		return err
	}
	defer restoreTerminal()
	// Use the alternate screen buffer so every frame replaces the previous one
	// and the user's shell history is restored intact on exit.
	if _, err := fmt.Fprint(out, "\x1b[?1049h\x1b[?7l\x1b[2J\x1b[H\x1b[?25l"); err != nil {
		return err
	}
	client := api.New(cfg)
	store := state.New(cfg)
	controller := screen.New(time.Now(), cfg.Rotation, cfg.AlertDuration)
	renderer := render.New(cfg)
	fetchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go fetchLoop(fetchCtx, client, store, cfg.Refresh)
	frameInterval := 250 * time.Millisecond
	if halfBlink := cfg.AlertBlink / 2; halfBlink < frameInterval {
		frameInterval = halfBlink
	}
	if frameInterval < 50*time.Millisecond {
		frameInterval = 50 * time.Millisecond
	}
	tick := time.NewTicker(frameInterval)
	defer tick.Stop()
	defer fmt.Fprint(out, "\x1b[0m\x1b[?7h\x1b[?25h\x1b[?1049l")
	lastFrame := ""
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-tick.C:
			w, h := terminal.Size()
			if cfg.Width > 0 && cfg.Height > 0 {
				w, h = cfg.Width, cfg.Height
			}
			snap := store.Get()
			view := screen.View{Kind: screen.Overview, Started: now}
			if !cfg.SingleMode {
				view = controller.Current(now, snap)
			}
			frame := renderer.Render(view, snap, w, h, now)
			if frame == lastFrame {
				continue
			}
			if _, err := fmt.Fprint(out, frame); err != nil {
				return err
			}
			lastFrame = frame
		}
	}
}
func fetchLoop(ctx context.Context, c *api.Client, s *state.Store, every time.Duration) {
	fetch := func() {
		ds, err := c.Fetch(ctx)
		if err != nil {
			s.Failure(err)
		} else {
			s.Success(ds, time.Now())
		}
	}
	fetch()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fetch()
		}
	}
}
