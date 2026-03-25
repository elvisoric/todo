package cmd

import (
	"fmt"
	"os"
)

var colorsEnabled = true

func init() {
	// Disable colors if NO_COLOR env is set or output is not a terminal
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
		return
	}
	fi, err := os.Stdout.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		colorsEnabled = false
	}
}

const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	dim       = "\033[2m"
	italic    = "\033[3m"
	red       = "\033[31m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	white     = "\033[37m"
	boldCyan  = "\033[1;36m"
	boldGreen = "\033[1;32m"
	boldRed   = "\033[1;31m"
)

func c(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return fmt.Sprintf("%s%s%s", color, text, reset)
}
