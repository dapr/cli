//go:build windows
// +build windows

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

package syscall

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/dapr/cli/pkg/print"

	"github.com/kolesnikovae/go-winjob"
	"github.com/kolesnikovae/go-winjob/jobapi"
)

var jbObj *winjob.JobObject

func SetupShutdownNotify(sigCh chan os.Signal) {
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

// CreateProcessGroupID creates a process group ID for the current process.
func CreateProcessGroupID() {
	// This is a no-op on windows.
	// Process group ID is not used for killing all the processes on windows.
	// Instead, we use combination of named event and job object to kill all the processes.
}

// AttachJobObjectToProcess attaches the process to a job object.
// It creates the job object if it doesn't exist.
func AttachJobObjectToProcess(jobName string, proc *os.Process) {
	if jbObj != nil {
		err := jbObj.Assign(proc)
		if err != nil {
			print.WarningStatusEvent(os.Stdout, "failed to assign process to job object: %s", err.Error())
		}
		return
	}
	jbObj, err := winjob.Create(jobName)
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "failed to create job object: %s", err.Error())
		return
	}
	// Below lines control the relation between Job object and processes attached to it.
	// By passing JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE flag, it will make sure that when
	// job object is closed all the processed must also be exited.
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	err = jobapi.SetInformationJobObject(jbObj.Handle, jobapi.JobObjectExtendedLimitInformation, unsafe.Pointer(&info), uint32(unsafe.Sizeof(info)))
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "failed to set job object info: %s", err.Error())
		return
	}
	err = jbObj.Assign(proc)
	if err != nil {
		print.WarningStatusEvent(os.Stdout, "failed to assign process to job object: %s", err.Error())
	}
}
