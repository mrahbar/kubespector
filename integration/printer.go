package integration

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/fatih/color"
	"os"
	"strings"
	"unicode/utf8"
	"bytes"
)

const (
	noType      = ""
	okType      = "[OK]"
	errType     = "[ERROR]"
	skippedType = "[SKIPPED]"
	warnType    = "[WARNING]"
	unknownType = "[UNKNOWN]"
	ignoredType = "[IGNORED]"
	infoType    = "[INFO]"
	debugType   = "[DEBUG]"
)

const tabWidth = 104
var out io.Writer = os.Stdout

var Green = color.New(color.FgGreen)
var Red = color.New(color.FgRed)
var Orange = color.New(color.FgRed, color.FgYellow)
var Blue = color.New(color.FgCyan)
var Magenta = color.New(color.FgMagenta)
var Yellow = color.New(color.FgYellow)
var White = color.New(color.FgHiWhite)

// PrettyPrintOk [OK](Green) with formatted string
func PrettyPrintOk(msg string, a ...interface{}) {
	printMsg(msg, okType, a...)
}

// PrettyPrintErr [ERROR](Red) with formatted string
func PrettyPrintErr(msg string, a ...interface{}) {
	printMsg(msg, errType, a...)
}

// PrettyPrint no type will be displayed, used for just single line printing
func PrettyPrint(msg string, a ...interface{}) {
	printMsg(msg, noType, a...)
}

func PrettyNewLine() {
	printMsg("", noType)
}

// PrettyPrintWarn [WARNING](Orange) with formatted string
func PrettyPrintWarn(msg string, a ...interface{}) {
	printMsg(msg, warnType, a...)
}

// PrettyPrintIgnored [IGNORED](Red) with formatted string
func PrettyPrintIgnored(msg string, a ...interface{}) {
	printMsg(msg, ignoredType, a...)
}

// PrettyPrintInfo [INFO] with formatted string
func PrettyPrintInfo(msg string, a ...interface{}) {
	printMsg(msg, infoType, a...)
}

// PrettyPrintDebug [DEBUG] with formatted string
func PrettyPrintDebug(msg string, a ...interface{}) {
	printMsg(msg, debugType, a...)
}

// PrettyPrintUnknown [UNKNOWN](Red) with formatted string
func PrettyPrintUnknown(msg string, a ...interface{}) {
	printMsg(msg, unknownType, a...)
}

// PrettyPrintSkipped [SKIPPED](blue) with formatted string
func PrettyPrintSkipped(msg string, a ...interface{}) {
	printMsg(msg, skippedType, a...)
}

// PrintOk print whole message in green(Red) format
func PrintOk(out io.Writer) {
	PrintColor(Green, okType)
}

// PrintOkln print whole message in green(Red) format
func PrintOkln(out io.Writer) {
	PrintColor(Green, okType+"\n")
}

// PrintError print whole message in error(Red) format
func PrintError(out io.Writer) {
	PrintColor(Red, errType)
}

// PrintWarn print whole message in warn(Orange) format
func PrintWarn(out io.Writer) {
	PrintColor(Orange, warnType)
}

// PrintSkipped print whole message in green(Red) format
func PrintSkipped(out io.Writer) {
	PrintColor(Blue, skippedType)
}

// PrintHeader will print header with predefined width
func PrintHeader(msg string, padding byte) {
	w := tabwriter.NewWriter(out, tabWidth, 0, 0, padding, 0)
	fmt.Fprintln(w, "")
	format := msg + "\t\n"
	fmt.Fprintf(w, format)
	w.Flush()
}

// PrintColor prints text in color
func PrintColor(clr *color.Color, msg string, a ...interface{}) {
	// Remove any newline, results in only one \n
	line := fmt.Sprintf("%s", clr.SprintfFunc()(msg, a...))
	fmt.Fprint(out, line)
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
		case errType, unknownType:
			clr = Red
		case warnType, ignoredType:
			clr = Orange
		case infoType:
			clr = Blue
		case skippedType:
			clr = Magenta
		case debugType:
			clr = Yellow
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
