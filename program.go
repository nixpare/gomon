package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type Program struct {
	dir string
	execName string
	args  []string
	exitC 	chan struct{}
	waiting int
	exec    *exec.Cmd
	redirect bool
	lastExitCode int
}

func NewProgram(dir string, redirect bool, execName string, args ...string) (*Program, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("exec: directory not found")
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("exec: dir is not a directory")
	}

	p := &Program{
		dir:      dir,
		execName: execName,
		args:     args,
		exitC:    make(chan struct{}, 3),
		redirect: redirect,
	}

	return p, nil
}

func (p *Program) IsRunning() bool {
	return p.exec != nil
}

func (p *Program) Start() error {
	if p.IsRunning() {
		return fmt.Errorf("program %s already running", p.execName)
	}

	err := p.start()
	if err != nil {
		return fmt.Errorf("error starting program %s: %v", p.execName, err)
	}

	return nil
}

func (p *Program) start() error {
	p.exec = exec.Command(p.execName, p.args...)
	if p.dir != "" {
		p.exec.Dir = p.dir
	}

	if p.redirect {
		p.exec.Stdout = os.Stdout
		p.exec.Stderr = os.Stderr
	}

	err := p.exec.Start()
	if err != nil {
		p.exec = nil
		return err
	}

	go p.wait()
	return nil
}

func (p *Program) wait() {
	if p.exec == nil {
		return
	}

	p.exec.Wait()
	
	p.lastExitCode = p.exec.ProcessState.ExitCode()
	p.exec = nil

	for p.waiting > 0 {
		p.exitC <- struct{}{}
		p.waiting --
	}
}

func (p *Program) Wait() {
	if p.exec == nil {
		return
	}

	p.waiting ++
	<- p.exitC
}

func(p *Program) Run() error {
	err := p.Start()
	if err != nil {
		return err
	}

	p.Wait()
	return nil
}

func (p *Program) Kill() {
	if p.exec != nil {
		p.exec.Process.Kill()
		p.Wait()
	}
}

func (p *Program) Stop() error {
	if p.exec == nil {
		return nil
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		p.Wait()
		wg.Done()
	}()

	SendCtrlC(p.exec.Process.Pid)
	wg.Wait()

	err := RestoreConsoleCtrlHandler()
	if err != nil {
		return err
	}

	return nil
}

func (p *Program) String() string {
	var state string
	if p.IsRunning() {
		state = fmt.Sprintf("Running - %d", p.exec.Process.Pid)
	} else {
		state = "Stopped"
	}
	return fmt.Sprintf("%s (%s)", p.execName, state)
}