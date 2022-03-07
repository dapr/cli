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

package rundata

/*
 * WARNING: This is a basic and temporary file based implementation to handle local state
 * and currently does not yet support multiple process concurrency or the ability to clean
 * up stale data from processes that did not gracefully shutdown. The local state file is
 * not used anymore. This code is still important to make sure that file is deleted on
 * uninstall.
 */

import (
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
		return err
	}

	return nil
}

func tryGetRunDataLock() (*lockfile.Lockfile, error) {
	lockFile, err := lockfile.New(filepath.Join(os.TempDir(), runDataLockFile))
	if err != nil {
		// TODO: Log once we implement logging.
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
