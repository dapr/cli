/*
Copyright 2022 The Dapr Authors
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

package standalone

import "os"

// must is a helper function that executes a function and expects it to succeed.
func must(f func() (string, error), message string) {
	if _, err := f(); err != nil {
		panic(err.Error() + ": " + message)
	}
}

// isSlimMode returns true if DAPR_E2E_INIT_SLIM is set to true.
func isSlimMode() bool {
	return os.Getenv("DAPR_E2E_INIT_SLIM") == "true"
}
