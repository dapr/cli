// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package kubernetes

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/dapr/dapr/pkg/apis/configuration/v1alpha1"
)

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
			expectedOutput: "  NAME       TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED              \n  appConfig  false            false            0s   " + formattedNow + "  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
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
			expectedOutput: "  NAME       TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED              \n  appConfig  false            false            0s   " + formattedNow + "  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "daprsystem",
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
			expectedOutput: "  NAME       TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED              \n  appConfig  false            false            0s   " + formattedNow + "  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
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
			expectedOutput: "  NAME  TRACING-ENABLED  METRICS-ENABLED  AGE  CREATED  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "not config",
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
			expectedOutput: "name: appConfig\nspec:\n  httppipelinespec:\n    handlers: []\n  tracingspec:\n    samplingrate: \"\"\n    zipkin:\n      endpointaddresss: \"\"\n  metricspec:\n    enabled: false\n  mtlsspec:\n    enabled: false\n    workloadcertttl: \"\"\n    allowedclockskew: \"\"\n  secrets:\n    scopes: []\n  accesscontrolspec:\n    defaultAction: \"\"\n    trustDomain: \"\"\n    policies: []\n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
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
			expectedOutput: "- name: appConfig1\n  spec:\n    httppipelinespec:\n      handlers: []\n    tracingspec:\n      samplingrate: \"\"\n      zipkin:\n        endpointaddresss: \"\"\n    metricspec:\n      enabled: false\n    mtlsspec:\n      enabled: false\n      workloadcertttl: \"\"\n      allowedclockskew: \"\"\n    secrets:\n      scopes: []\n    accesscontrolspec:\n      defaultAction: \"\"\n      trustDomain: \"\"\n      policies: []\n- name: appConfig2\n  spec:\n    httppipelinespec:\n      handlers: []\n    tracingspec:\n      samplingrate: \"\"\n      zipkin:\n        endpointaddresss: \"\"\n    metricspec:\n      enabled: false\n    mtlsspec:\n      enabled: false\n      workloadcertttl: \"\"\n      allowedclockskew: \"\"\n    secrets:\n      scopes: []\n    accesscontrolspec:\n      defaultAction: \"\"\n      trustDomain: \"\"\n      policies: []\n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig1",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig2",
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
			expectedOutput: "{\n  \"name\": \"appConfig\",\n  \"spec\": {\n    \"httpPipeline\": {\n      \"handlers\": null\n    },\n    \"tracing\": {\n      \"samplingRate\": \"\",\n      \"zipkin\": {\n        \"endpointAddress\": \"\"\n      }\n    },\n    \"metric\": {\n      \"enabled\": false\n    },\n    \"mtls\": {\n      \"enabled\": false,\n      \"workloadCertTTL\": \"\",\n      \"allowedClockSkew\": \"\"\n    },\n    \"secrets\": {\n      \"scopes\": null\n    },\n    \"accessControl\": {\n      \"defaultAction\": \"\",\n      \"trustDomain\": \"\",\n      \"policies\": null\n    }\n  }\n}",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
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
			expectedOutput: "[\n  {\n    \"name\": \"appConfig1\",\n    \"spec\": {\n      \"httpPipeline\": {\n        \"handlers\": null\n      },\n      \"tracing\": {\n        \"samplingRate\": \"\",\n        \"zipkin\": {\n          \"endpointAddress\": \"\"\n        }\n      },\n      \"metric\": {\n        \"enabled\": false\n      },\n      \"mtls\": {\n        \"enabled\": false,\n        \"workloadCertTTL\": \"\",\n        \"allowedClockSkew\": \"\"\n      },\n      \"secrets\": {\n        \"scopes\": null\n      },\n      \"accessControl\": {\n        \"defaultAction\": \"\",\n        \"trustDomain\": \"\",\n        \"policies\": null\n      }\n    }\n  },\n  {\n    \"name\": \"appConfig2\",\n    \"spec\": {\n      \"httpPipeline\": {\n        \"handlers\": null\n      },\n      \"tracing\": {\n        \"samplingRate\": \"\",\n        \"zipkin\": {\n          \"endpointAddress\": \"\"\n        }\n      },\n      \"metric\": {\n        \"enabled\": false\n      },\n      \"mtls\": {\n        \"enabled\": false,\n        \"workloadCertTTL\": \"\",\n        \"allowedClockSkew\": \"\"\n      },\n      \"secrets\": {\n        \"scopes\": null\n      },\n      \"accessControl\": {\n        \"defaultAction\": \"\",\n        \"trustDomain\": \"\",\n        \"policies\": null\n      }\n    }\n  }\n]",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Configuration{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig1",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ConfigurationSpec{},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig2",
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
