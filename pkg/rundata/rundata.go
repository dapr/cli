// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package rundata

/*
 * WARNING: This is a basic and temporary file based implementation to handle local state
 * and currently does not yet support multiple process concurrency or the ability to clean
 * up stale data from processes that did not gracefully shutdown. It is expected an out-of
 * -process implementation will eventually replace this one.
 */

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nightlyone/lockfile"
)

var (
	runDataFile     string = "dapr-run-data.ldj"
	runDataLockFile string = "dapr-run-data.lock"
)

type RunData struct {
	DaprRunID    string
	DaprHTTPPort int
	DaprGRPCPort int
	AppID        string
	AppPort      int
	Command      string
	Created      time.Time
	PID          int
}

func AppendRunData(runData *RunData) error {
	lockFile, err := tryGetRunDataLock()
	if err != nil {
		return err
	}

	defer lockFile.Unlock()

	runDataFilePath := filepath.Join(os.TempDir(), runDataFile)
	runDataFile, err := os.OpenFile(runDataFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer runDataFile.Close()

	err = appendRunDataEntry(runDataFile, runData)
	if err != nil {
		return err
	}

	return nil
}

func ReadAllRunData() (*[]RunData, error) {
	lockFile, err := tryGetRunDataLock()
	if err != nil {
		return nil, err
	}

	defer lockFile.Unlock()

	runData := []RunData{}

	runFilePath := filepath.Join(os.TempDir(), runDataFile)
	runFileData, err := ioutil.ReadFile(runFilePath)
	if err != nil {
		return &runData, nil
	}

	runDataJSON := strings.Split(string(runFileData), "\n")

	for _, lineJSON := range runDataJSON {
		var line RunData
		err = json.Unmarshal([]byte(lineJSON), &line)
		if err != nil {
			// Ignore broken lines for now
			continue
		}
		runData = append(runData, line)
	}

	return &runData, nil
}

func ClearRunData(daprRunID string) error {
	lockFile, err := tryGetRunDataLock()
	if err != nil {
		return err
	}

	defer lockFile.Unlock()

	runFilePath := filepath.Join(os.TempDir(), runDataFile)
	runFileData, err := ioutil.ReadFile(runFilePath)
	if err != nil {
		return err
	}

	runDataJSON := strings.Split(string(runFileData), "\n")

	runFile, err := os.Create(runFilePath)
	if err != nil {
		return err
	}

	defer runFile.Close()

	for _, lineJSON := range runDataJSON {
		var line RunData
		err = json.Unmarshal([]byte(lineJSON), &line)
		if err != nil {
			// Ignore broken lines for now
			continue
		}
		if line.DaprRunID != daprRunID {
			appendRunDataEntry(runFile, &line)
			// Ignore errors for now
		}
	}

	return nil
}

func appendRunDataEntry(runDataFile *os.File, runData *RunData) error {
	runDataJSON, err := json.Marshal(runData)
	if err != nil {
		return err
	}

	_, err = runDataFile.Write(runDataJSON)
	if err != nil {
		return err
	}

	_, err = runDataFile.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func tryGetRunDataLock() (*lockfile.Lockfile, error) {
	lockFile, err := lockfile.New(filepath.Join(os.TempDir(), runDataLockFile))
	if err != nil {
		// TODO: Log once we implement logging
		return nil, err
	}

	for i := 0; i < 10; i++ {
		err = lockFile.TryLock()

		// Error handling is essential, as we only try to get the lock.
		if err == nil {
			return &lockFile, nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil, err
}
