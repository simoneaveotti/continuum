package main

import (
	"fmt"
	"time"

	"continuum/internal/prompt"
)

var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// progressReporter turns a stream of phase messages into visible activity:
// on a terminal it animates a spinner next to the current phase and checks
// it off once the next phase starts (or finish is called); when stdout
// isn't a terminal (piped/logged) it falls back to one static line per
// phase, since an animated spinner would just emit garbage.
type progressReporter struct {
	stopCurrent func()
}

func newProgressReporter() *progressReporter {
	return &progressReporter{}
}

func (p *progressReporter) report(msg string) {
	if p.stopCurrent != nil {
		p.stopCurrent()
		p.stopCurrent = nil
	}
	if !prompt.IsInteractiveOutput() {
		fmt.Println(msg)
		return
	}
	p.stopCurrent = startSpinner(msg)
}

// finish checks off whatever phase is still spinning. Call after the last
// progress message, once the overall operation has completed.
func (p *progressReporter) finish() {
	if p.stopCurrent != nil {
		p.stopCurrent()
		p.stopCurrent = nil
	}
}

// startSpinner animates msg with a spinner until the returned func is called,
// then leaves a checkmark in its place.
func startSpinner(msg string) func() {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-stopCh:
				fmt.Printf("\r\x1b[K%s ✓\n", msg)
				return
			case <-ticker.C:
				fmt.Printf("\r\x1b[K%s %c", msg, spinnerFrames[i%len(spinnerFrames)])
				i++
			}
		}
	}()

	return func() {
		close(stopCh)
		<-doneCh
	}
}
