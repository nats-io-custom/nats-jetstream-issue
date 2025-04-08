package cobra_utils

import "fmt"

// ANSI color codes
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

// Style modifiers
const (
	Bold      = "\033[1m"
	Underline = "\033[4m"
)

// Printer provides methods for colored console output
type Printer struct {
	EnableColors bool
}

// NewPrinter creates a new Printer instance
func NewPrinter() *Printer {
	return &Printer{
		EnableColors: true,
	}
}

// Print prints text in the specified color
func (p *Printer) Print(color string, text string) {
	if p.EnableColors {
		print(color + text + Reset)
	} else {
		print(text)
	}
	print("\n")
}

// Printf prints formatted text in the specified color
func (p *Printer) Printf(color string, format string, a ...interface{}) {
	text := fmt.Sprintf(format, a...)
	if p.EnableColors {
		print(color + text + Reset)
	} else {
		print(text)
	}
	print("\n")
}

// Println prints text in the specified color with a newline
func (p *Printer) Println(color string, text string) {
	if p.EnableColors {
		println(color + text + Reset)
	} else {
		println(text)
	}

}

// PrintBold prints bold text in the specified color
func (p *Printer) PrintBold(color string, text string) {
	if p.EnableColors {
		print(Bold + color + text + Reset)
	} else {
		print(text)
	}
	print("\n")
}

// PrintfBold prints formatted bold text in the specified color
func (p *Printer) PrintfBold(color string, format string, a ...interface{}) {
	text := fmt.Sprintf(format, a...)
	if p.EnableColors {
		print(Bold + color + text + Reset)
	} else {
		print(text)
	}
	print("\n")
}

// Success prints text in green
func (p *Printer) Success(text string) {
	p.Println(Green, text)
}

// Successf prints formatted text in green
func (p *Printer) Successf(format string, a ...interface{}) {
	p.Printf(Green, format, a...)
}

// Error prints text in red
func (p *Printer) Error(text string) {
	p.Println(Red, text)
}

// Errorf prints formatted text in red
func (p *Printer) Errorf(format string, a ...interface{}) {
	p.Printf(Red, format, a...)
}

// Warning prints text in yellow
func (p *Printer) Warning(text string) {
	p.Println(Yellow, text)
}

// Warningf prints formatted text in yellow
func (p *Printer) Warningf(format string, a ...interface{}) {
	p.Printf(Yellow, format, a...)
}

// Info prints text in blue
func (p *Printer) Info(text string) {
	p.Println(Blue, text)
}

// Infof prints formatted text in blue
func (p *Printer) Infof(format string, a ...interface{}) {
	p.Printf(Blue, format, a...)
}
