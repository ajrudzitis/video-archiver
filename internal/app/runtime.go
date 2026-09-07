package app

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

// Runtime carries handles that are only available after the tea.Program is
// constructed, so background goroutines can send messages into the program.
type Runtime struct {
	program atomic.Pointer[tea.Program]
}

func (r *Runtime) SetProgram(p *tea.Program) { r.program.Store(p) }

func (r *Runtime) Send(msg tea.Msg) {
	if p := r.program.Load(); p != nil {
		p.Send(msg)
	}
}
