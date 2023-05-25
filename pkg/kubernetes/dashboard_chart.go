/*
Copyright 2023 The Dapr Authors
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

import "github.com/Masterminds/semver/v3"

const daprHelmChartWithDashboard = "<= 1.10.x"

// IsDashboardIncluded returns true if dashboard is included in Helm chart version for Dapr.
func IsDashboardIncluded(runtimeVersion string) (bool, error) {
	c, err := semver.NewConstraint(daprHelmChartWithDashboard)
	if err != nil {
		return false, err
	}

	v, err := semver.NewVersion(runtimeVersion)
	if err != nil {
		return false, err
	}

	return c.Check(v), nil
}
