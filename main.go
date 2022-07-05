package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	startTime time.Time = time.Now()
	nExecs int = 0
)

func main() {
	fmt.Println(BlueString("GoMon started"))

	var dir string
	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else {
		dir, _ = os.Getwd()
	}

	tempExecDir := strings.TrimRight(os.TempDir(), "/\\")
	if runtime.GOOS == "windows" {
		tempExecDir += "\\gomon"
	} else {
		tempExecDir += "/gomon"
	}

	info, err := os.Stat(tempExecDir)
	if err != nil {
		err := os.Mkdir(tempExecDir, 0777)
		if err != nil {
			log.Fatalln(PrintError(err.Error()))
		}
	} else {
		if !info.IsDir() {
			log.Fatalln(RedString(fmt.Sprintf("cannot use directory %s because its already a file", tempExecDir)))
		}
	}

	tempExecName := "a-*"
	if runtime.GOOS == "windows" {
		tempExecName += ".exe"
	}

	tempExecFile, err := os.CreateTemp(tempExecDir, tempExecName)
	if err != nil {
		log.Fatalln(PrintError(err.Error()))
	}
	tempExecName = tempExecFile.Name()
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
	fmt.Println(BlueString("\nGoMon terminated"))

	os.Remove(tempExecName)
}

func goRunRoutine(execName, dir string, scheduler *Scheduler, waitForRecompile *bool) {
	nExecs ++
	secs := time.Since(startTime).Seconds()
	if secs > 0 {
		ratio := float64(nExecs) / secs
		if ratio > 0.8 {
			time.Sleep(time.Second)
		}
	} else {
		if nExecs > 1 {
			time.Sleep(time.Second)
		}
	}

	if *waitForRecompile {
		fmt.Println(YellowString("\n  •  Waiting for change before recompiling ...\n"))
		scheduler.WaitForChange()
		*waitForRecompile = false
	}

	fmt.Println(CyanString("\n  •  Building executable ..."))
	p, err := NewProgram(dir, true, "go", "build", "-o", execName)
	if err != nil {
		fmt.Println(PrintError(err.Error()))
		*waitForRecompile = true
		return
	}

	err = p.Run()
	if err != nil {
		fmt.Println(PrintError(err.Error()))
		*waitForRecompile = true
		return
	}

	fmt.Println(PrintExitCode(p.lastExitCode))
	if p.lastExitCode != 0 {
		*waitForRecompile = true
		return
	}

	fmt.Println(CyanString("\n  •  Running executable ..."))

	var args []string
	if len(os.Args) > 2 {
		args = append(args, os.Args[2:]...)
	}

	p, err = NewProgram(dir, true, execName, args...)
	if err != nil {
		log.Fatalln(PrintError(err.Error()))
	}

	p.Start()

	wg := new(sync.WaitGroup)

	go func() {
		scheduler.WaitForChange()
		wg.Add(1)

		err = p.Stop()
		if err != nil {
			log.Fatalln(PrintError(err.Error()))
		}
		wg.Done()
	}()

	p.Wait()
	wg.Wait()
	fmt.Println(PrintExitCode(p.lastExitCode))
}

func checkRoutine(dir string, scheduler *Scheduler) {
	matches, err := WalkMatch(dir, "*.go")
	if err != nil {
		log.Fatalln(PrintError(err.Error()))
	}

	filesInfo := InitFilesInfoMap(matches)

	for {
		time.Sleep(time.Second)
		
		matches, err := WalkMatch(dir, "*.go")
		if err != nil {
			log.Fatalln(PrintError(err.Error()))
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
