package test

import (
    "io"
    "fmt"
    "os"
)

type MockLogWriter struct {
    Out io.Writer
}

func (p *MockLogWriter) PrettyNewLine() {

}
func (p *MockLogWriter) PrintHeader(msg string, padding byte) {

}

// Print no type will be displayed, used for just single line printing
func (p *MockLogWriter) Print(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// LogWriterr [ERROR](Red) with formatted string
func (p *MockLogWriter) PrintCritical(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
    os.Exit(1)
}

// LogWriterr [ERROR](Red) with formatted string
func (p *MockLogWriter) PrintErr(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrintWarn [WARNING](Orange) with formatted string
func (p *MockLogWriter) PrintWarn(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrintIgnored [IGNORED](Red) with formatted string
func (p *MockLogWriter) PrintIgnored(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrettyPrintOk [OK](Green) with formatted string
func (p *MockLogWriter) PrintOk(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrintInfo [INFO] with formatted string
func (p *MockLogWriter) PrintInfo(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrintDebug [DEBUG] with formatted string
func (p *MockLogWriter) PrintDebug(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrintDebug [TRACE] with formatted string
func (p *MockLogWriter) PrintTrace(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrintUnknown [UNKNOWN](Red) with formatted string
func (p *MockLogWriter) PrintUnknown(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}

// PrintSkipped [SKIPPED](blue) with formatted string
func (p *MockLogWriter) PrintSkipped(msg string, a ...interface{}) {
    fmt.Fprintf(p.Out, msg+"\n", a...)
}
