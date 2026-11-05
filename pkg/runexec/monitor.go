/*
Copyright 2026 The Dapr Authors
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

package runexec

import "os"

// WaitForApps waits for every configured app to exit or for a shutdown signal.
// Each started app command must send exactly one result to appExited. Entries
// without a command are included in appCount but do not send a result, keeping
// their sidecars running until a signal arrives. The caller owns both channels
// and must leave them open while processes can still report their exits.
// The results indicate whether all apps exited and whether any app failed.
func WaitForApps(signals <-chan os.Signal, appExited <-chan error, appCount int) (bool, bool) {
	failed := false
	for {
		select {
		case err := <-appExited:
			failed = failed || err != nil
			appCount--
			if appCount == 0 {
				return true, failed
			}
		case <-signals:
			// A signal can race with already queued results. Preserve failures
			// reported before shutdown without waiting for still-running apps.
			for {
				select {
				case err := <-appExited:
					failed = failed || err != nil
				default:
					return false, failed
				}
			}
		}
	}
}
