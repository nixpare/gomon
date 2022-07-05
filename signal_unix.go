//go:build linux || darwin

package main

import (
	"os"
)

func SendCtrlC(p *os.Process) error {
	return p.Signal(os.Interrupt)
}

func RestoreConsoleCtrlHandler() error {
	return nil
}