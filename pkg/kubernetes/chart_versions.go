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

package kubernetes

// This can be removed in the future (>= 1.0) when chart versions align to runtime versions.
var chartVersionsMap = map[string]string{
	"0.7.0": "0.4.0",
	"0.7.1": "0.4.1",
	"0.8.0": "0.4.2",
	"0.9.0": "0.4.3",
}

// chartVersion will return the corresponding Helm Chart version for the given runtime version.
// If the specified version is not found, it is assumed that the chart version equals the runtime version.
func chartVersion(runtimeVersion string) string {
	v, ok := chartVersionsMap[runtimeVersion]
	if ok {
		return v
	}
	return runtimeVersion
}
