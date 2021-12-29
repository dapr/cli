// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dapr/cli/pkg/print"
	"golang.org/x/sys/windows"
)

func setupShutdownNotify(sigCh chan os.Signal) {
	//This will catch Ctrl-C
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Unlike Linux/Mac, you can't just send a SIGTERM from another process
	// In order for 'dapr stop' to be able to signal gracefully we must use a named event in Windows
	go func() {
		eventName, _ := syscall.UTF16FromString(fmt.Sprintf("dapr_cli_%v", os.Getpid()))
		eventHandle, _ := windows.CreateEvent(nil, 0, 0, &eventName[0])
		_, err := windows.WaitForSingleObject(eventHandle, windows.INFINITE)
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "Unable to wait for shutdown event. 'dapr stop' will not work. Error: %s", err.Error())
			return
		}
		sigCh <- os.Interrupt
	}()
}
