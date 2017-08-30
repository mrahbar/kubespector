package integration

import (
	"fmt"
	"io"
	"text/tabwriter"

	"bytes"
	"github.com/fatih/color"
	"os"
	"strings"
	"unicode/utf8"
)

const (
	noType      = ""
	okType      = "[OK]"
	criticalType     = "[CRITICAL]"
	errType     = "[ERROR]"
	skippedType = "[SKIPPED]"
	warnType    = "[WARNING]"
	unknownType = "[UNKNOWN]"
	ignoredType = "[IGNORED]"
	infoType    = "[INFO]"
	debugType   = "[DEBUG]"
    traceType   = "[TRACE]"
)

const tabWidth = 124

var out io.Writer = os.Stdout

var Green = color.New(color.FgGreen)
var Red = color.New(color.FgRed)
var Orange = color.New(color.FgRed, color.FgYellow)
var Blue = color.New(color.FgCyan)
var Magenta = color.New(color.FgMagenta)
var Yellow = color.New(color.FgYellow)
var White = color.New(color.FgHiWhite)
var Grey = color.New(color.FgWhite)

type LogWriter interface {
	PrintNewLine()
	PrintHeader(msg string, padding byte)
	Print(msg string, a ...interface{})
	PrintCritical(msg string, a ...interface{})
	PrintErr(msg string, a ...interface{})
	PrintWarn(msg string, a ...interface{})
	PrintIgnored(msg string, a ...interface{})
	PrintOk(msg string, a ...interface{})
	PrintInfo(msg string, a ...interface{})
	PrintDebug(msg string, a ...interface{})
	PrintTrace(msg string, a ...interface{})
	PrintUnknown(msg string, a ...interface{})
	PrintSkipped(msg string, a ...interface{})
}

type Printer struct {
    LogLevel LogLevel
}

// Print no type will be displayed, used for just single line printing
func (p *Printer) Print(msg string, a ...interface{}) {
    if p.LogLevel >= INFO {
        PrettyPrint(msg, a...)
    }
}

// PrintErr [ERROR](Red) with formatted string
func (p *Printer) PrintCritical(msg string, a ...interface{}) {
	if p.LogLevel >= CRITICAL {
		PrettyPrintCritical(msg, a...)
	}
	os.Exit(1)
}

// PrintErr [ERROR](Red) with formatted string
func (p *Printer) PrintErr(msg string, a ...interface{}) {
    if p.LogLevel >= ERROR {
        PrettyPrintErr(msg, a...)
    }
}

// PrintWarn [WARNING](Orange) with formatted string
func (p *Printer) PrintWarn(msg string, a ...interface{}) {
    if p.LogLevel >= WARNING {
        PrettyPrintWarn(msg, a...)
    }
}

// PrintIgnored [IGNORED](Red) with formatted string
func (p *Printer) PrintIgnored(msg string, a ...interface{}) {
    if p.LogLevel >= INFO {
		printMsg(msg, ignoredType, a...)
    }
}

// PrettyPrintOk [OK](Green) with formatted string
func (p *Printer) PrintOk(msg string, a ...interface{}) {
    if p.LogLevel >= INFO {
		printMsg(msg, okType, a...)
    }
}

// PrintInfo [INFO] with formatted string
func (p *Printer) PrintInfo(msg string, a ...interface{}) {
    if p.LogLevel >= INFO {
		printMsg(msg, infoType, a...)
    }
}

// PrintDebug [DEBUG] with formatted string
func (p *Printer) PrintDebug(msg string, a ...interface{}) {
    if p.LogLevel >= DEBUG {
		printMsg(msg, debugType, a...)
    }
}

// PrintDebug [TRACE] with formatted string
func (p *Printer) PrintTrace(msg string, a ...interface{}) {
    if p.LogLevel >= TRACE {
		printMsg(msg, traceType, a...)
    }
}

// PrintUnknown [UNKNOWN](Red) with formatted string
func (p *Printer) PrintUnknown(msg string, a ...interface{}) {
    if p.LogLevel >= INFO {
		printMsg(msg, unknownType, a...)
    }
}

// PrintSkipped [SKIPPED](blue) with formatted string
func (p *Printer) PrintSkipped(msg string, a ...interface{}) {
    if p.LogLevel >= INFO {
		printMsg(msg, skippedType, a...)
    }
}

func (p *Printer) PrintNewLine() {
    printMsg("", noType)
}

// PrintHeader will print header with predefined width
func (p *Printer) PrintHeader(msg string, padding byte) {
	w := tabwriter.NewWriter(out, tabWidth, 0, 0, padding, 0)
	fmt.Fprintln(w, "")
	format := msg + "\t\n"
	fmt.Fprintf(w, format)
	w.Flush()
}

// PrettyPrintOk [OK](Green) with formatted string
func PrettyPrintOk(msg string, a ...interface{}) {
	printMsg(msg, okType, a...)
}

// PrettyPrintErr [CRITICAL](Red) with formatted string
func PrettyPrintCritical(msg string, a ...interface{}) {
	printMsg(msg, criticalType, a...)
}

// PrettyPrintErr [ERROR](Red) with formatted string
func PrettyPrintErr(msg string, a ...interface{}) {
	printMsg(msg, errType, a...)
}

// PrettyPrint no type will be displayed, used for just single line printing
func PrettyPrint(msg string, a ...interface{}) {
	printMsg(msg, noType, a...)
}

// PrettyPrintWarn [WARNING](Orange) with formatted string
func PrettyPrintWarn(msg string, a ...interface{}) {
	printMsg(msg, warnType, a...)
}

func printMsg(msg, status string, a ...interface{}) {
	width := tabWidth - 4
	w := tabwriter.NewWriter(out, width, 0, 0, ' ', 0)

	var msgBuffer bytes.Buffer
	fmt.Fprintf(&msgBuffer, msg, a...)
	carriageReturnSplits := strings.Split(msgBuffer.String(), "\n")
	msg = ""
	for _, cr := range carriageReturnSplits {
		crFormated := formatToTab(cr, width)
		if msg != "" {
			msg = fmt.Sprintf("%s\n%s", msg, crFormated)
		} else {
			msg = crFormated
		}
	}

	// print message
	fmt.Fprintf(w, strings.TrimFunc(msg, func(r rune) bool { return r == ' ' })+"\t")

	// print status
	if status != noType {
		// get correct color
		var clr *color.Color
		switch status {
		case okType:
			clr = Green
		case errType, unknownType, criticalType:
			clr = Red
		case warnType, ignoredType:
			clr = Orange
		case infoType:
			clr = Blue
		case skippedType:
			clr = Magenta
		case debugType:
			clr = Yellow
        case traceType:
            clr = Grey
		}

		fmt.Fprintf(w, "%s\n", clr.SprintFunc()(status))

	} else {
		fmt.Fprint(w, "\n")
	}

	w.Flush()
}

func formatToTab(msg string, width int) string {
	if utf8.RuneCountInString(msg) > width {
		msgSplits := strings.Split(msg, " ")
		msg = ""
		msgLine := ""

		for _, split := range msgSplits {
			newLine := fmt.Sprintf("%s %s", msgLine, split)
			if len(newLine) < width {
				msgLine = newLine
			} else {
				if msg != "" {
					msg = fmt.Sprintf("%s\n%s", msg, msgLine)
				} else {
					msg = msgLine
				}

				msgLine = split
			}
		}
		if msgLine != "" {
			msg = fmt.Sprintf("%s\n%s", msg, msgLine)
		}
	}

	return msg
}
