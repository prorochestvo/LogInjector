package terminal

import (
	"os"
	"runtime"
)

// Color codes
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
)

// Check if terminal supports colors.
// This function may not correctly recognize color support.
func isColorTerminal() bool {
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}
	return runtime.GOOS != "windows"
}

// Colorize text with the given color.
// If the terminal does not support colors, the text will be returned as is.
// Example:
// fmt.Println(Colorize(Red, "The TEXT is red"))
func Colorize(text string, color string) string {
	if isColorTerminalCache == nil {
		b := isColorTerminal()
		isColorTerminalCache = &b
	}

	if isColorTerminalCache != nil && *isColorTerminalCache == true {
		return color + text + Reset
	} else {
		return text
	}
}

var isColorTerminalCache *bool = nil
