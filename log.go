package main

import (
	"fmt"
)

const (
	DEFAULT_TERMINAL = "\x1b[0m"
	GREY_TERMINAL = "\x1b[38;5;8m"
	RED_TERMINAL = "\x1b[38;5;9m"
	GREEN_TERMINAL = "\x1b[38;5;10m"
	YELLOW_TERMINAL = "\x1b[38;5;11m"
	BLUE_TERMINAL = "\x1b[38;5;12m"
	CYAN_TERMINAL = "\x1b[38;5;14m"
)

func GreyString(s string) string {
	return GREY_TERMINAL + s + DEFAULT_TERMINAL
}

func RedString(s string) string {
	return RED_TERMINAL + s + DEFAULT_TERMINAL
}

func GreenString(s string) string {
	return GREEN_TERMINAL + s + DEFAULT_TERMINAL
}

func YellowString(s string) string {
	return YELLOW_TERMINAL + s + DEFAULT_TERMINAL
}

func BlueString(s string) string {
	return BLUE_TERMINAL + s + DEFAULT_TERMINAL
}

func CyanString(s string) string {
	return CYAN_TERMINAL + s + DEFAULT_TERMINAL
}

func PrintExitCode(pid int) string {
	var color string
	if pid == 0 {
		color = GREEN_TERMINAL
	} else {
		color = RED_TERMINAL
	}
	return fmt.Sprintf("%s  -  Exited with code %d%s", color, pid, DEFAULT_TERMINAL)
}

func PrintError(s string) string {
	return RedString("\n  •  " + s + "\n")
}
