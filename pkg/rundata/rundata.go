// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package rundata

/*
 * WARNING: This is a basic and temporary file based implementation to handle local state
 * and currently does not yet support multiple process concurrency or the ability to clean
 * up stale data from processes that did not gracefully shutdown. The local state file is
 * not used anymore. This code is still important to make sure that file is deleted on
 * uninstall.
 */

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nightlyone/lockfile"
)

var (
	runDataFile     = "dapr-run-data.ldj"
	runDataLockFile = "dapr-run-data.lock"
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

// DeleteRunDataFile deletes the deprecated RunData file.
func DeleteRunDataFile() error {
	lockFile, err := tryGetRunDataLock()
	if err != nil {
		return err
	}
	defer lockFile.Unlock()

	runFilePath := filepath.Join(os.TempDir(), runDataFile)
	err = os.Remove(runFilePath)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	return nil
}

func tryGetRunDataLock() (*lockfile.Lockfile, error) {
	lockFile, err := lockfile.New(filepath.Join(os.TempDir(), runDataLockFile))
	if err != nil {
		// TODO: Log once we implement logging
		return nil, fmt.Errorf("error: %w", err)
	}

	for i := 0; i < 10; i++ {
		err = lockFile.TryLock()

		// Error handling is essential, as we only try to get the lock.
		if err == nil {
			return &lockFile, nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil, fmt.Errorf("error: %w", err)
}
