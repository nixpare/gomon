package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	fmt.Println("GoMon started")

	var dir string
	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}

	tempExecDir := os.TempDir() + "\\gomon"

	info, err := os.Stat(tempExecDir)
	if err != nil {
		err := os.Mkdir(tempExecDir, 0777)
		if err != nil {
			log.Fatalln(err)
		}
	}

	if !info.IsDir() {
		log.Fatalf("cannot use directory %s because its already a file\n", tempExecDir)
	}

	tempExecFile, err := os.CreateTemp(tempExecDir, "a-*.exe")
	if err != nil {
		log.Fatalln(err)
	}
	tempExecName := tempExecFile.Name()
	tempExecFile.Close()

	scheduler := NewScheduler()

	go checkRoutine(dir, scheduler)

	exitC := make(chan os.Signal, 3)
	signal.Notify(exitC, os.Interrupt, syscall.SIGTERM)

	wg := new(sync.WaitGroup)
	waitForRecompile := new(bool)

	wg.Add(1)
	go func() {
		goRunRoutine(tempExecName, dir, scheduler, waitForRecompile)
		wg.Done()

		for {
			time.Sleep(time.Millisecond * 100)

			wg.Add(1)
			goRunRoutine(tempExecName, dir, scheduler, waitForRecompile)
			wg.Done()
		}
	}()

	<-exitC
	wg.Wait()
	fmt.Println("GoMon terminated")

	os.Remove(tempExecName)
}

func goRunRoutine(execName, dir string, scheduler *Scheduler, waitForRecompile *bool) {
	if *waitForRecompile {
		fmt.Println("Waiting for change before recompiling ...")
		scheduler.WaitForChange()
		*waitForRecompile = false
	}

	fmt.Println("Building executable ...")
	p, err := NewProgram(dir, true, "go", "build", "-o", execName)
	if err != nil {
		fmt.Println(err)
		*waitForRecompile = true
		return
	}

	err = p.Run()
	if err != nil {
		fmt.Println(err)
		*waitForRecompile = true
		return
	}

	fmt.Printf("\nBuild operation terminated with status code %d\n", p.lastExitCode)
	if p.lastExitCode != 0 {
		*waitForRecompile = true
		return
	}

	fmt.Println("Running executable ...")

	var args []string
	if len(os.Args) > 2 {
		args = append(args, os.Args[2:]...)
	}

	p, err = NewProgram(dir, true, execName, args...)
	if err != nil {
		log.Fatalln(err)
	}

	p.Start()

	wg := new(sync.WaitGroup)

	go func() {
		scheduler.WaitForChange()
		wg.Add(1)

		err = p.Stop()
		if err != nil {
			log.Fatalln(err)
		}
		wg.Done()
	}()

	p.Wait()
	wg.Wait()
	fmt.Printf("Executable terminated with status code %d\n", p.lastExitCode)
}

func checkRoutine(dir string, scheduler *Scheduler) {
	matches, err := WalkMatch(dir, "*.go")
	if err != nil {
		log.Fatalln(err)
	}

	filesInfo := InitFilesInfoMap(matches)

	for {
		time.Sleep(time.Second)
		
		matches, err := WalkMatch(dir, "*.go")
		if err != nil {
			log.Fatalln(err)
		}

		checkedMap := make(map[string]bool)
		for key := range filesInfo {
			checkedMap[key] = false
		}

		var triggerRestart bool

		for _, path := range matches {
			newFileInfo, err := os.Stat(path)
			if err != nil {
				continue
			}

			fileInfo, ok := filesInfo[path]
			if !ok {
				triggerRestart = true
				filesInfo[path] = newFileInfo

				continue
			}

			if newFileInfo.ModTime() != fileInfo.ModTime() {
				triggerRestart = true
				filesInfo[path] = newFileInfo
			}

			checkedMap[path] = true
		}

		for key, value := range checkedMap {
			if !value {
				delete(filesInfo, key)
			}
		}

		if triggerRestart {
			scheduler.TriggerChange()
		}
	}
}
