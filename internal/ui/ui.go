package ui

import (
	"fmt"
	"os"
)

// Color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
)

func Success(msg string) {
	fmt.Fprintf(os.Stderr, "%s✓%s %s\n", Green, Reset, msg)
}

func Warning(msg string) {
	fmt.Fprintf(os.Stderr, "%s⚠%s %s\n", Yellow, Reset, msg)
}

func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s✗%s %s\n", Red, Reset, msg)
}

func Info(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s\n", Cyan, msg, Reset)
}

func Headerf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "\n%s%s%s\n", Bold, fmt.Sprintf(format, a...), Reset)
}

func Successf(format string, a ...interface{}) {
	Success(fmt.Sprintf(format, a...))
}

func Warningf(format string, a ...interface{}) {
	Warning(fmt.Sprintf(format, a...))
}

func Errorf(format string, a ...interface{}) {
	Error(fmt.Sprintf(format, a...))
}

func DiffAdd(line string) {
	fmt.Fprintf(os.Stdout, "%s+ %s%s\n", Green, line, Reset)
}

func DiffRemove(line string) {
	fmt.Fprintf(os.Stdout, "%s- %s%s\n", Red, line, Reset)
}

func DiffChange(line string) {
	fmt.Fprintf(os.Stdout, "%s~ %s%s\n", Yellow, line, Reset)
}
