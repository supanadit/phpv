package terminal

import (
	"fmt"
	"os"
	"time"

	"github.com/supanadit/phpv/assembler"
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
func newProgressReporter() (assembler.ProgressFunc, func()) {
	progressCh := make(chan progressMsg, 64)
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)
		var current string
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for {
			select {
			case msg, ok := <-progressCh:
				if !ok {
					return
				}
				glyph := stageGlyph(msg.stage)
				if msg.stage == "done" || msg.stage == "skip" {
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s\n", glyph, msg.message)
				} else {
					current = fmt.Sprintf("%s %s", glyph, msg.message)
					fmt.Fprintf(os.Stdout, "\r\033[2K%s %s", spinnerFrames[frame%len(spinnerFrames)], current)
				}
			case <-ticker.C:
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
