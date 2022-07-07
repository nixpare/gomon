package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/alessio-pareto/goutils"
)

func compileRoutine(dir, execName string, s *Scheduler) bool {
	fmt.Println(YellowString("\n  •  Building executable ..."))

	p, err := goutils.NewProgram(dir, true, "go", "build", "-o", execName)
	if err != nil {
		fmt.Println(PrintError(err.Error()))
		s.waitChangesForRecompile = true
		return true
	}

	err = p.Run()
	if err != nil {
		fmt.Println(PrintError(err.Error()))
		s.waitChangesForRecompile = true
		return true
	}

	fmt.Println(PrintExitCode(p.LastExitCode()))
	if p.LastExitCode() != 0 {
		s.waitChangesForRecompile = true
		return true
	}

	return false
}

func runRoutine(dir, execName string, s *Scheduler) {
	fmt.Println(LightBlueString("\n  •  Running executable ..."))

	var args []string
	if len(os.Args) > 2 {
		args = append(args, os.Args[2:]...)
	}

	p, err := goutils.NewProgram(dir, true, execName, args...)
	if err != nil {
		log.Fatalln(PrintError(err.Error()))
	}

	p.Start()

	wg := new(sync.WaitGroup)
	wg.Add(2)

	var exited, exitedForChange bool

	go func() {
		s.WaitForChange(&exited)
		if exited {
			wg.Done()
			return
		}

		err = p.Stop()
		if err != nil {
			log.Fatalln(PrintError(err.Error()))
		}

		if !exited {
			exited = true
			exitedForChange = true
			wg.Done()
		}
	}()

	go func() {
		p.Wait()
		if !exited {
			exited = true
		}
		wg.Done()
	}()
	
	wg.Wait()
	fmt.Println(PrintExitCode(p.LastExitCode()))

	if !exitedForChange {
		s.waitChangesForRecompile = true
	}
}