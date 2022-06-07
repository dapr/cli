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

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/dapr/cli/pkg/print"
)

func setupShutdownNotify(sigCh chan os.Signal) {
	// This will catch Ctrl-C.
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Unlike Linux/Mac, you can't just send a SIGTERM from another process.
	// In order for 'dapr stop' to be able to signal gracefully we must use a named event in Windows.
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
