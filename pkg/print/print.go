// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package print

import (
	"fmt"
	"io"
	"runtime"

	"github.com/fatih/color"
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

// SuccessStatusEvent reports on a success event.
func SuccessStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "✅  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// FailureStatusEvent reports on a failure event.
func FailureStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "❌  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// PendingStatusEvent reports on a pending event.
func PendingStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "⌛  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}

// InfoStatusEvent reports status information on an event.
func InfoStatusEvent(w io.Writer, fmtstr string, a ...interface{}) {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(w, "%s\n", fmt.Sprintf(fmtstr, a...))
	} else {
		fmt.Fprintf(w, "ℹ️  %s\n", fmt.Sprintf(fmtstr, a...))
	}
}
