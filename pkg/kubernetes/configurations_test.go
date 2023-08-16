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
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
)

//nolint:dupword
func TestConfigurations(t *testing.T) {
	now := meta_v1.Now()
	formattedNow := now.Format("2006-01-02 15:04.05")
	testCases := []struct {
		name           string
		configName     string
		outputFormat   string
		expectedOutput string
		errString      string
		errorExpected  bool
		k8sConfig      []v1alpha1.Configuration
	}{
		{
			name:           "List one config",
			configName:     "",
			outputFormat:   "",
			expectedOutput: "  NAMESPACE  NAME       TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED              \n  default    appConfig  false            false            0s   " + formattedNow + "  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
		{
			name:           "Error on fetching config",
			configName:     "",
			outputFormat:   "",
			expectedOutput: "",
			errString:      "could not fetch config",
			errorExpected:  true,
			k8sConfig:      []v1alpha1.Configuration{},
		},
		{
			name:           "Filters out daprsystem",
			configName:     "",
			outputFormat:   "",
			expectedOutput: "  NAMESPACE  NAME       TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED              \n  default    appConfig  false            false            0s   " + formattedNow + "  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "daprsystem",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
		{
			name:           "Name does match",
			configName:     "appConfig",
			outputFormat:   "list",
			expectedOutput: "  NAMESPACE  NAME       TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED              \n  default    appConfig  false            false            0s   " + formattedNow + "  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
		{
			name:           "Name does not match",
			configName:     "appConfig",
			outputFormat:   "list",
			expectedOutput: "  NAMESPACE  NAME  TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "not config",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
		{
			name:           "Yaml one config",
			configName:     "",
			outputFormat:   "yaml",
			expectedOutput: "- name: appConfig\n  namespace: default\n  spec:\n    apphttppipelinespec:\n      handlers: []\n    httppipelinespec:\n      handlers: []\n    tracingspec:\n      samplingrate: \"\"\n      stdout: false\n      zipkin:\n        endpointaddresss: \"\"\n      otel:\n        protocol: \"\"\n        endpointAddress: \"\"\n        isSecure: false\n    metricspec:\n      enabled: false\n      rules: []\n    metricsspec:\n      enabled: false\n      rules: []\n    mtlsspec:\n      enabled: false\n      workloadcertttl: \"\"\n      allowedclockskew: \"\"\n    secrets:\n      scopes: []\n    accesscontrolspec:\n      defaultAction: \"\"\n      trustDomain: \"\"\n      policies: []\n    nameresolutionspec:\n      component: \"\"\n      version: \"\"\n      configuration:\n        json:\n          raw: []\n    features: []\n    apispec:\n      allowed: []\n      denied: []\n    componentsspec: {}\n    loggingspec:\n      apiLogging:\n        enabled: false\n        obfuscateURLs: false\n        omitHealthChecks: false\n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
		{
			name:           "Yaml two configs",
			configName:     "",
			outputFormat:   "yaml",
			expectedOutput: "- name: appConfig1\n  namespace: default\n  spec:\n    apphttppipelinespec:\n      handlers: []\n    httppipelinespec:\n      handlers: []\n    tracingspec:\n      samplingrate: \"\"\n      stdout: false\n      zipkin:\n        endpointaddresss: \"\"\n      otel:\n        protocol: \"\"\n        endpointAddress: \"\"\n        isSecure: false\n    metricspec:\n      enabled: false\n      rules: []\n    metricsspec:\n      enabled: false\n      rules: []\n    mtlsspec:\n      enabled: false\n      workloadcertttl: \"\"\n      allowedclockskew: \"\"\n    secrets:\n      scopes: []\n    accesscontrolspec:\n      defaultAction: \"\"\n      trustDomain: \"\"\n      policies: []\n    nameresolutionspec:\n      component: \"\"\n      version: \"\"\n      configuration:\n        json:\n          raw: []\n    features: []\n    apispec:\n      allowed: []\n      denied: []\n    componentsspec: {}\n    loggingspec:\n      apiLogging:\n        enabled: false\n        obfuscateURLs: false\n        omitHealthChecks: false\n- name: appConfig2\n  namespace: default\n  spec:\n    apphttppipelinespec:\n      handlers: []\n    httppipelinespec:\n      handlers: []\n    tracingspec:\n      samplingrate: \"\"\n      stdout: false\n      zipkin:\n        endpointaddresss: \"\"\n      otel:\n        protocol: \"\"\n        endpointAddress: \"\"\n        isSecure: false\n    metricspec:\n      enabled: false\n      rules: []\n    metricsspec:\n      enabled: false\n      rules: []\n    mtlsspec:\n      enabled: false\n      workloadcertttl: \"\"\n      allowedclockskew: \"\"\n    secrets:\n      scopes: []\n    accesscontrolspec:\n      defaultAction: \"\"\n      trustDomain: \"\"\n      policies: []\n    nameresolutionspec:\n      component: \"\"\n      version: \"\"\n      configuration:\n        json:\n          raw: []\n    features: []\n    apispec:\n      allowed: []\n      denied: []\n    componentsspec: {}\n    loggingspec:\n      apiLogging:\n        enabled: false\n        obfuscateURLs: false\n        omitHealthChecks: false\n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig1",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig2",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
		{
			name:           "Json one config",
			configName:     "",
			outputFormat:   "json",
			expectedOutput: "[\n  {\n    \"name\": \"appConfig\",\n    \"namespace\": \"default\",\n    \"spec\": {\n      \"appHttpPipeline\": {\n        \"handlers\": null\n      },\n      \"httpPipeline\": {\n        \"handlers\": null\n      },\n      \"tracing\": {\n        \"samplingRate\": \"\",\n        \"stdout\": false,\n        \"zipkin\": {\n          \"endpointAddress\": \"\"\n        },\n        \"otel\": {\n          \"protocol\": \"\",\n          \"endpointAddress\": \"\",\n          \"isSecure\": false\n        }\n      },\n      \"metric\": {\n        \"enabled\": false,\n        \"rules\": null\n      },\n      \"metrics\": {\n        \"enabled\": false,\n        \"rules\": null\n      },\n      \"mtls\": {\n        \"enabled\": false,\n        \"workloadCertTTL\": \"\",\n        \"allowedClockSkew\": \"\"\n      },\n      \"secrets\": {\n        \"scopes\": null\n      },\n      \"accessControl\": {\n        \"defaultAction\": \"\",\n        \"trustDomain\": \"\",\n        \"policies\": null\n      },\n      \"nameResolution\": {\n        \"component\": \"\",\n        \"version\": \"\",\n        \"configuration\": null\n      },\n      \"api\": {},\n      \"components\": {},\n      \"logging\": {\n        \"apiLogging\": {\n          \"enabled\": false,\n          \"obfuscateURLs\": false,\n          \"omitHealthChecks\": false\n        }\n      }\n    }\n  }\n]",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
		{
			name:           "Json two configs",
			configName:     "",
			outputFormat:   "json",
			expectedOutput: "[\n  {\n    \"name\": \"appConfig1\",\n    \"namespace\": \"default\",\n    \"spec\": {\n      \"appHttpPipeline\": {\n        \"handlers\": null\n      },\n      \"httpPipeline\": {\n        \"handlers\": null\n      },\n      \"tracing\": {\n        \"samplingRate\": \"\",\n        \"stdout\": false,\n        \"zipkin\": {\n          \"endpointAddress\": \"\"\n        },\n        \"otel\": {\n          \"protocol\": \"\",\n          \"endpointAddress\": \"\",\n          \"isSecure\": false\n        }\n      },\n      \"metric\": {\n        \"enabled\": false,\n        \"rules\": null\n      },\n      \"metrics\": {\n        \"enabled\": false,\n        \"rules\": null\n      },\n      \"mtls\": {\n        \"enabled\": false,\n        \"workloadCertTTL\": \"\",\n        \"allowedClockSkew\": \"\"\n      },\n      \"secrets\": {\n        \"scopes\": null\n      },\n      \"accessControl\": {\n        \"defaultAction\": \"\",\n        \"trustDomain\": \"\",\n        \"policies\": null\n      },\n      \"nameResolution\": {\n        \"component\": \"\",\n        \"version\": \"\",\n        \"configuration\": null\n      },\n      \"api\": {},\n      \"components\": {},\n      \"logging\": {\n        \"apiLogging\": {\n          \"enabled\": false,\n          \"obfuscateURLs\": false,\n          \"omitHealthChecks\": false\n        }\n      }\n    }\n  },\n  {\n    \"name\": \"appConfig2\",\n    \"namespace\": \"default\",\n    \"spec\": {\n      \"appHttpPipeline\": {\n        \"handlers\": null\n      },\n      \"httpPipeline\": {\n        \"handlers\": null\n      },\n      \"tracing\": {\n        \"samplingRate\": \"\",\n        \"stdout\": false,\n        \"zipkin\": {\n          \"endpointAddress\": \"\"\n        },\n        \"otel\": {\n          \"protocol\": \"\",\n          \"endpointAddress\": \"\",\n          \"isSecure\": false\n        }\n      },\n      \"metric\": {\n        \"enabled\": false,\n        \"rules\": null\n      },\n      \"metrics\": {\n        \"enabled\": false,\n        \"rules\": null\n      },\n      \"mtls\": {\n        \"enabled\": false,\n        \"workloadCertTTL\": \"\",\n        \"allowedClockSkew\": \"\"\n      },\n      \"secrets\": {\n        \"scopes\": null\n      },\n      \"accessControl\": {\n        \"defaultAction\": \"\",\n        \"trustDomain\": \"\",\n        \"policies\": null\n      },\n      \"nameResolution\": {\n        \"component\": \"\",\n        \"version\": \"\",\n        \"configuration\": null\n      },\n      \"api\": {},\n      \"components\": {},\n      \"logging\": {\n        \"apiLogging\": {\n          \"enabled\": false,\n          \"obfuscateURLs\": false,\n          \"omitHealthChecks\": false\n        }\n      }\n    }\n  }\n]",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig1",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig2",
						Namespace:         "default",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buff bytes.Buffer
			err := writeConfigurations(&buff,
				func() (*v1alpha1.ConfigurationList, error) {
					if len(tc.errString) > 0 {
						return nil, fmt.Errorf(tc.errString)
					}

					return &v1alpha1.ConfigurationList{Items: tc.k8sConfig}, nil
				}, tc.configName, tc.outputFormat)
			if tc.errorExpected {
				assert.Error(t, err, "expected an error")
				assert.Equal(t, tc.errString, err.Error(), "expected error strings to match")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.Equal(t, tc.expectedOutput, buff.String(), "expected output strings to match")
			}
		})
	}
}
