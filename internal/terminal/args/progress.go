package args

import (
	"fmt"
	"os"
	"time"

	"github.com/supanadit/phpv/pkg/progress"
)

// progressMsg is sent by the assembler through a progress callback.
type progressMsg struct {
	stage   string
	message string
}

// spinnerFrames are the animation frames for the spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// newProgressReporter starts a spinner on stdout and returns the callback to
// hand to the assembler plus a stop function that waits for the spinner to
// finish. Callers that want raw build output (--verbose) pass nil instead.
//
// When stdout is not a terminal (CI, piped logs) the spinner is disabled:
// each progress message is written once as a plain line, so logs stay clean
// instead of getting one redraw line per 80ms tick.
func newProgressReporter() (progress.Func, func()) {
	progressCh := make(chan progressMsg, 64)
	doneCh := make(chan struct{})
	interactive := stdoutIsTerminal()

	go func() {
		defer close(doneCh)
		var current string
		var ticker *time.Ticker
		var ticks <-chan time.Time
		if interactive {
			ticker = time.NewTicker(80 * time.Millisecond)
			defer ticker.Stop()
			ticks = ticker.C
		}
		frame := 0
		for {
			select {
			case msg, ok := <-progressCh:
				if !ok {
					return
				}
				glyph := stageGlyph(msg.stage)
				if msg.stage == "done" || msg.stage == "skip" || !interactive {
					fmt.Fprintf(os.Stdout, "%s %s\n", glyph, msg.message)
				} else {
					current = fmt.Sprintf("%s %s", glyph, msg.message)
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
				}
			case <-ticks:
				if current != "" {
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
				}
				frame++
			}
		}
	}()

	return func(stage, message string) {
			progressCh <- progressMsg{stage: stage, message: message}
		}, func() {
			close(progressCh)
			<-doneCh
		}
}

// stdoutIsTerminal reports whether stdout is an interactive terminal that can
// handle \r-based spinner redraws. CI runners (CI env var set) never get the
// spinner, since every redraw becomes a separate log line.
func stdoutIsTerminal() bool {
	if os.Getenv("CI") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// stageGlyph returns a short glyph for the given stage.
func stageGlyph(stage string) string {
	switch stage {
	case "resolve":
		return "→"
	case "deps":
		return "→"
	case "download":
		return "↓"
	case "prepare":
		return "⚙"
	case "build":
		return "⚙"
	case "configure":
		return "⚙"
	case "make":
		return "⚙"
	case "install":
		return "↑"
	case "skip":
		return "↷"
	case "patch":
		return "✎"
	case "error":
		return "✗"
	case "done":
		return "✓"
	default:
		return "·"
	}
}
