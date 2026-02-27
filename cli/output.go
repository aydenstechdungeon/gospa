package cli

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
	bold    = "\033[1m"
	dim     = "\033[2m"
)

// ColorPrinter provides colored output utilities
type ColorPrinter struct {
	useColor bool
}

// NewColorPrinter creates a new color printer
func NewColorPrinter() *ColorPrinter {
	// Check if stdout is a terminal
	useColor := isTerminal()
	return &ColorPrinter{useColor: useColor}
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func (p *ColorPrinter) colorize(color, text string) string {
	if !p.useColor {
		return text
	}
	return color + text + reset
}

// Success prints a green success message with checkmark
func (p *ColorPrinter) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", p.colorize(green, "✓"), msg)
}

// Error prints a red error message with X mark
func (p *ColorPrinter) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", p.colorize(red, "✗"), msg)
}

// Warning prints a yellow warning message
func (p *ColorPrinter) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", p.colorize(yellow, "!"), msg)
}

// Info prints a blue info message
func (p *ColorPrinter) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", p.colorize(blue, "→"), msg)
}

// Step prints a step in a process
func (p *ColorPrinter) Step(step int, total int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	prefix := fmt.Sprintf("[%d/%d]", step, total)
	fmt.Printf("%s %s\n", p.colorize(cyan, prefix), msg)
}

// Title prints a bold title
func (p *ColorPrinter) Title(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("\n%s\n\n", p.colorize(bold, msg))
}

// Subtitle prints a dimmed subtitle
func (p *ColorPrinter) Subtitle(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s\n", p.colorize(dim, msg))
}

// Bold returns bold text
func (p *ColorPrinter) Bold(text string) string {
	return p.colorize(bold, text)
}

// Green returns green text
func (p *ColorPrinter) Green(text string) string {
	return p.colorize(green, text)
}

// Red returns red text
func (p *ColorPrinter) Red(text string) string {
	return p.colorize(red, text)
}

// Yellow returns yellow text
func (p *ColorPrinter) Yellow(text string) string {
	return p.colorize(yellow, text)
}

// Cyan returns cyan text
func (p *ColorPrinter) Cyan(text string) string {
	return p.colorize(cyan, text)
}

// Dim returns dimmed text
func (p *ColorPrinter) Dim(text string) string {
	return p.colorize(dim, text)
}

// ProgressBar displays a simple progress bar
func (p *ColorPrinter) ProgressBar(current, total int, label string) {
	width := 30
	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r%s [%s] %d%% %s", label, p.colorize(green, bar), int(percent*100), p.Dim("..."))
	if current >= total {
		fmt.Println()
	}
}

// Spinner shows a spinning animation
type Spinner struct {
	frames  []string
	current int
	printer *ColorPrinter
	message string
}

// NewSpinner creates a new spinner
func NewSpinner(printer *ColorPrinter, message string) *Spinner {
	return &Spinner{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		printer: printer,
		message: message,
	}
}

// Tick advances the spinner
func (s *Spinner) Tick() {
	frame := s.frames[s.current%len(s.frames)]
	fmt.Printf("\r%s %s ", s.printer.colorize(cyan, frame), s.message)
	s.current++
}

// Done stops the spinner with success
func (s *Spinner) Done() {
	fmt.Printf("\r%s %s\n", s.printer.colorize(green, "✓"), s.message)
}

// Fail stops the spinner with failure
func (s *Spinner) Fail() {
	fmt.Printf("\r%s %s\n", s.printer.colorize(red, "✗"), s.message)
}

// PrintBanner prints the GoSPA banner
func PrintBanner() {
	printer := NewColorPrinter()
	banner := `
   ___  ____  _____ 
  / _ \/ __ \/ ___/ 
 / /_/ / /_/ / /     
 \__,_/ .___/_/      
     /_/             	
`
	fmt.Println(printer.colorize(cyan, banner))
	fmt.Printf("%s %s\n", printer.Dim("A modern SPA framework for Go"), printer.Dim("v0.1.4.1"))
	fmt.Println()
}
