package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var (
	startTime time.Time = time.Now()
	nExecs int = 0
	exiting bool
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

	s := NewScheduler()

	go checkRoutine(dir, s)

	exitC := make(chan os.Signal, 3)
	signal.Notify(exitC, os.Interrupt, syscall.SIGTERM)

	go func() {
		for !exiting {
			goRunRoutine(tempExecName, dir, s)
		}
	}()

	<-exitC
	exiting = true

	s.RoutineWG.Wait()
	fmt.Println(BlueString("\nGoMon terminated"))

	os.Remove(tempExecName)
}

func goRunRoutine(execName, dir string, s *Scheduler) {
	nExecs ++
	secs := time.Since(startTime).Seconds()
	if secs > 0 {
		ratio := float64(nExecs) / secs
		if ratio > 0.5 {
			time.Sleep(time.Second * 2)
		} else {
			startTime = time.Now()
		}
	} else {
		if nExecs > 1 {
			time.Sleep(time.Second)
		} else {
			startTime = time.Now()
		}
	}

	if s.waitChangesForRecompile {
		fmt.Println(OrangeString("\n  •  Waiting for change before recompiling ...\n"))
		s.WaitForChange(nil)
		s.waitChangesForRecompile = false
	}

	defer time.Sleep(time.Millisecond * 100)

	s.RoutineWG.Add(1)
	mustReturn := compileRoutine(dir, execName, s)
	s.RoutineWG.Done()

	if mustReturn {
		return
	}

	s.RoutineWG.Add(1)
	runRoutine(dir, execName, s)
	s.RoutineWG.Done()
}

func checkRoutine(dir string, s *Scheduler) {
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
			s.TriggerChange()
		}
	}
}
