/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
//nolint
package print

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

const (
	windowsOS = "windows"
)

type logStatus string

const (
	LogSuccess logStatus = "success"
	LogFailure logStatus = "failure"
	LogWarning logStatus = "warning"
	LogInfo    logStatus = "info"
	LogPending logStatus = "pending"
)

type Result bool

const (
	Success Result = true
	Failure Result = false
)

var (
	Yellow    = color.New(color.FgHiYellow, color.Bold).SprintFunc()
	Green     = color.New(color.FgHiGreen, color.Bold).SprintFunc()
	Blue      = color.New(color.FgHiBlue, color.Bold).SprintFunc()
	Cyan      = color.New(color.FgCyan, color.Bold, color.Underline).SprintFunc()
	Red       = color.New(color.FgHiRed, color.Bold).Add(color.Italic).SprintFunc()
	White     = color.New(color.FgWhite).SprintFunc()
	WhiteBold = color.New(color.FgWhite, color.Bold).SprintFunc()
)

var logAsJSON bool

func EnableJSONFormat() {
	logAsJSON = true
}

func IsJSONLogEnabled() bool {
	return logAsJSON
}

// StatusEvent reports a event log with given status.
func StatusEvent(w io.Writer, status logStatus, fmtstr string, a ...any) {
	if logAsJSON {
		logJSON(w, string(status), fmt.Sprintf(fmtstr, a...))
		return
	}
	if (w != os.Stdout && w != os.Stderr) || runtime.GOOS == windowsOS {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
		return
	}
	switch status {
	case LogSuccess:
		fmt.Fprintf(w, "✅  %s\n", fmt.Sprintf(fmtstr, a...))
	case LogFailure:
		fmt.Fprintf(w, "❌  %s\n", fmt.Sprintf(fmtstr, a...))
	case LogWarning:
		fmt.Fprintf(w, "⚠  %s\n", fmt.Sprintf(fmtstr, a...))
	case LogPending:
		fmt.Fprintf(w, "⌛  %s\n", fmt.Sprintf(fmtstr, a...))
	case LogInfo:
		fmt.Fprintf(w, "ℹ️  %s\n", fmt.Sprintf(fmtstr, a...))
	default:
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// SuccessStatusEvent reports on a success event.
func SuccessStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if logAsJSON {
		logJSON(w, string(LogSuccess), fmt.Sprintf(fmtstr, a...))
	} else if runtime.GOOS == windowsOS {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "✅  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// FailureStatusEvent reports on a failure event.
func FailureStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if logAsJSON {
		logJSON(w, string(LogFailure), fmt.Sprintf(fmtstr, a...))
	} else if runtime.GOOS == windowsOS {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "❌  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// WarningStatusEvent reports on a failure event.
func WarningStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if logAsJSON {
		logJSON(w, string(LogWarning), fmt.Sprintf(fmtstr, a...))
	} else if runtime.GOOS == windowsOS {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "⚠  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// PendingStatusEvent reports on a pending event.
func PendingStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if logAsJSON {
		logJSON(w, string(LogPending), fmt.Sprintf(fmtstr, a...))
	} else if runtime.GOOS == windowsOS {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "⌛  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// InfoStatusEvent reports status information on an event.
func InfoStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if logAsJSON {
		logJSON(w, string(LogInfo), fmt.Sprintf(fmtstr, a...))
	} else if runtime.GOOS == windowsOS {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "ℹ️  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

func Spinner(w io.Writer, fmtstr string, a ...interface{}) func(result Result) {
	msg := fmt.Sprintf(fmtstr, a...)
	var once sync.Once
	var s *spinner.Spinner

	if logAsJSON {
		logJSON(w, string(LogPending), msg)
	} else if runtime.GOOS == windowsOS {
		fmt.Fprintf(w, "%s\n", msg)

		return func(Result) {} // Return a dummy func
	} else {
		s = spinner.New(spinner.CharSets[0], 100*time.Millisecond)
		s.Writer = w
		s.Color("cyan")
		s.Suffix = fmt.Sprintf("  %s", msg)
		s.Start()
	}

	return func(result Result) {
		once.Do(func() {
			if s != nil {
				s.Stop()
			}
			if result {
				SuccessStatusEvent(w, msg)
			} else {
				FailureStatusEvent(w, msg)
			}
		})
	}
}

func logJSON(w io.Writer, status, message string) {
	type jsonLog struct {
		Time    time.Time `json:"time"`
		Status  string    `json:"status"`
		Message string    `json:"msg"`
	}

	l := jsonLog{
		Time:    time.Now().UTC(),
		Status:  status,
		Message: message,
	}
	jsonBytes, err := json.Marshal(&l)
	if err != nil {
		// Fall back on printing the simple message without JSON.
		// This is unlikely.
		fmt.Fprintln(w, message)

		return
	}

	fmt.Fprintf(w, "%s\n", string(jsonBytes))
}

type CustomLogWriter struct {
	W io.Writer
}

func (c CustomLogWriter) Write(p []byte) (int, error) {
	write := func(w io.Writer, isStdIO bool) (int, error) {
		b := p
		if !isStdIO {
			// below regex is used to replace the color codes from the logs collected in the log file.
			reg := regexp.MustCompile("\x1b\\[[\\d;]+m")
			b = reg.ReplaceAll(b, []byte(""))
		}
		n, err := w.Write(b)
		if err != nil {
			return n, err
		}
		if n != len(b) {
			return n, io.ErrShortWrite
		}
		return len(b), nil
	}
	wIface := reflect.ValueOf(c.W).Interface()
	switch wType := wIface.(type) {
	case *os.File:
		if wType == os.Stderr || wType == os.Stdout {
			return write(c.W, true)
		} else {
			return write(c.W, false)
		}
	default:
		return write(c.W, false)
	}
}
