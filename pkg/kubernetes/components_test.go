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

	v1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
)

func TestComponents(t *testing.T) {
	now := meta_v1.Now()
	formattedNow := now.Format("2006-01-02 15:04.05")
	testCases := []struct {
		name           string
		configName     string
		outputFormat   string
		expectedOutput string
		errString      string
		errorExpected  bool
		k8sConfig      []v1alpha1.Component
	}{
		{
			name:           "List one config",
			configName:     "",
			outputFormat:   "",
			expectedOutput: "  NAME       TYPE         VERSION  SCOPES  CREATED              AGE  \n  appConfig  state.redis  v1               " + formattedNow + "  0s   \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
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
			k8sConfig:      []v1alpha1.Component{},
		},
		{
			name:           "Filters out daprsystem",
			configName:     "",
			outputFormat:   "",
			expectedOutput: "  NAME       TYPE         VERSION  SCOPES  CREATED              AGE  \n  appConfig  state.redis  v1               " + formattedNow + "  0s   \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "daprsystem",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{},
				},
			},
		},
		{
			name:           "Name does match",
			configName:     "appConfig",
			outputFormat:   "list",
			expectedOutput: "  NAME       TYPE         VERSION  SCOPES  CREATED              AGE  \n  appConfig  state.redis  v1               " + formattedNow + "  0s   \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
			},
		},
		{
			name:           "Name does not match",
			configName:     "appConfig",
			outputFormat:   "list",
			expectedOutput: "  NAME  TYPE  VERSION  SCOPES  CREATED  AGE  \n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "not config",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
			},
		},
		{
			name:           "Yaml one config",
			configName:     "",
			outputFormat:   "yaml",
			expectedOutput: "name: appConfig\nspec:\n  type: state.redis\n  version: v1\n  ignoreerrors: false\n  metadata: []\n  inittimeout: \"\"\n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
			},
		},
		{
			name:           "Yaml two configs",
			configName:     "",
			outputFormat:   "yaml",
			expectedOutput: "- name: appConfig1\n  spec:\n    type: state.redis\n    version: v1\n    ignoreerrors: false\n    metadata: []\n    inittimeout: \"\"\n- name: appConfig2\n  spec:\n    type: state.redis\n    version: v1\n    ignoreerrors: false\n    metadata: []\n    inittimeout: \"\"\n",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig1",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig2",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
			},
		},
		{
			name:           "Json one config",
			configName:     "",
			outputFormat:   "json",
			expectedOutput: "{\n  \"name\": \"appConfig\",\n  \"spec\": {\n    \"type\": \"state.redis\",\n    \"version\": \"v1\",\n    \"ignoreErrors\": false,\n    \"metadata\": null,\n    \"initTimeout\": \"\"\n  }\n}",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
			},
		},
		{
			name:           "Json two configs",
			configName:     "",
			outputFormat:   "json",
			expectedOutput: "[\n  {\n    \"name\": \"appConfig1\",\n    \"spec\": {\n      \"type\": \"state.redis\",\n      \"version\": \"v1\",\n      \"ignoreErrors\": false,\n      \"metadata\": null,\n      \"initTimeout\": \"\"\n    }\n  },\n  {\n    \"name\": \"appConfig2\",\n    \"spec\": {\n      \"type\": \"state.redis\",\n      \"version\": \"v1\",\n      \"ignoreErrors\": false,\n      \"metadata\": null,\n      \"initTimeout\": \"\"\n    }\n  }\n]",
			errString:      "",
			errorExpected:  false,
			k8sConfig: []v1alpha1.Component{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig1",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "appConfig2",
						CreationTimestamp: now,
					},
					Spec: v1alpha1.ComponentSpec{
						Type:    "state.redis",
						Version: "v1",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buff bytes.Buffer
			err := writeComponents(&buff,
				func() (*v1alpha1.ComponentList, error) {
					if len(tc.errString) > 0 {
						return nil, fmt.Errorf(tc.errString)
					}

					return &v1alpha1.ComponentList{Items: tc.k8sConfig}, nil
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
