//go:build !windows
// +build !windows

// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"os"
	"os/signal"
	"syscall"
)

func setupShutdownNotify(sigCh chan os.Signal) {
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
}
