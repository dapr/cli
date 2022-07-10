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

import (
	"encoding/json"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dapr/cli/utils"
	v1alpha1 "github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
)

func GetDefaultConfiguration() v1alpha1.Configuration {
	return v1alpha1.Configuration{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "daprsystem",
		},
		Spec: v1alpha1.ConfigurationSpec{
			MTLSSpec: v1alpha1.MTLSSpec{
				Enabled:          true,
				WorkloadCertTTL:  "24h",
				AllowedClockSkew: "15m",
			},
		},
	}
}

func GetDaprControlPlaneCurrentConfig() (*v1alpha1.Configuration, error) {
	namespace, err := GetDaprNamespace()
	if err != nil {
		return nil, err
	}
	output, err := utils.RunCmdAndWait("kubectl", "get", "configurations/daprsystem", "-n", namespace, "-o", "json")
	if err != nil {
		return nil, err
	}
	var config v1alpha1.Configuration
	json.Unmarshal([]byte(output), &config)
	return &config, nil
}
