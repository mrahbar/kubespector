package integration

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/fatih/color"
	"os"
)

const (
	noType      = ""
	okType      = "[OK]"
	errType     = "[ERROR]"
	skippedType = "[SKIPPED]"
	warnType    = "[WARNING]"
	unknownType = "[UNKNOWN]"
	ignoredType = "[IGNORED]"
)

var out io.Writer = os.Stdout

var Green = color.New(color.FgGreen)
var Red = color.New(color.FgRed)
var Orange = color.New(color.FgRed, color.FgYellow)
var Blue = color.New(color.FgCyan)
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

// PrettyPrintWarn [WARNING](Orange) with formatted string
func PrettyPrintWarn(msg string, a ...interface{}) {
	printMsg(msg, warnType, a...)
}

// PrettyPrintIgnored [IGNORED](Red) with formatted string
func PrettyPrintIgnored(msg string, a ...interface{}) {
	printMsg(msg, ignoredType, a...)
}

// PrettyPrintUnknown [UNREACHABLE](Red) with formatted string
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
	w := tabwriter.NewWriter(out, 104, 0, 0, padding, 0)
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
	w := tabwriter.NewWriter(out, 100, 0, 0, ' ', 0)
	// print message
	format := msg + "\t"
	fmt.Fprintf(w, format, a...)

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
		case skippedType:
			clr = Blue
		}

		sformat := "%s\n"
		fmt.Fprintf(w, sformat, clr.SprintFunc()(status))

	} else {
		fmt.Fprint(w, "\n")
	}

	w.Flush()
}
