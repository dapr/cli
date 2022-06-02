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

package standalone

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDetails(t *testing.T) {
	correctDetails := `{
		"daprd" : "1.7.0",
		"dashboard": "0.10.0",
		"cli": "1.7.0",
		"daprBinarySubDir": "dist",
		"dockerImageSubDir": "docker",
		"daprImageName": "daprio/dapr:1.7.2",
		"daprImageFileName": "daprio-dapr-1.7.2.tar.gz"
	}`
	f, err := os.CreateTemp("", "*-details.json")
	if err != nil {
		t.Fatalf("error creating temp directory for testing: %s", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(correctDetails)
	f.Close()
	bd := bundleDetails{}
	err = bd.readAndParseDetails(f.Name())
	assert.NoError(t, err, "expected no error on parsing correct details in file")
	assert.Equal(t, "1.7.0", *bd.RuntimeVersion, "expected versions to match")
	assert.Equal(t, "0.10.0", *bd.DashboardVersion, "expected versions to match")
	assert.Equal(t, "dist", *bd.BinarySubDir, "expected value to match")
	assert.Equal(t, "docker", *bd.ImageSubDir, "expected value to match")
	assert.Equal(t, "daprio/dapr:1.7.2", bd.getPlacementImageName(), "expected value to match")
	assert.Equal(t, "daprio-dapr-1.7.2.tar.gz", bd.getPlacementImageFileName(), "expected value to match")
}

func TestParseDetailsMissingDetails(t *testing.T) {
	missingDetails := `{
		"daprd" : "1.7.0",
		"dashboard": "0.10.0",
		"cli": "1.7.0",
		"daprImageName": "daprio/dapr:1.7.2"
		"daprImageFileName": "daprio-dapr-1.7.2.tar.gz"
	}`
	f, err := os.CreateTemp("", "*-details.json")
	if err != nil {
		t.Fatalf("error creating temp directory for testing: %s", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(missingDetails)
	f.Close()
	bd := bundleDetails{}
	err = bd.readAndParseDetails(f.Name())
	assert.Error(t, err, "expected error on parsing missing details in file")
}

func TestParseDetailsEmptyDetails(t *testing.T) {
	missingDetails := `{
		"daprd" : "",
		"dashboard": "",
		"cli": "1.7.0",
		"daprBinarySubDir": "dist",
		"dockerImageSubDir": "docker",
		"daprImageName": "daprio/dapr:1.7.2",
		"daprImageFileName": "daprio-dapr-1.7.2.tar.gz"
	}`
	f, err := os.CreateTemp("", "*-details.json")
	if err != nil {
		t.Fatalf("error creating temp directory for testing: %s", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(missingDetails)
	f.Close()
	bd := bundleDetails{}
	err = bd.readAndParseDetails(f.Name())
	assert.Error(t, err, "expected error on parsing missing details in file")
}

func TestParseDetailsMissingFile(t *testing.T) {
	f, err := os.CreateTemp("", "*-details.json")
	if err != nil {
		t.Fatalf("error creating temp directory for testing: %s", err)
	}
	f.Close()
	os.Remove(f.Name())
	bd := bundleDetails{}
	err = bd.readAndParseDetails(f.Name())
	assert.Error(t, err, "expected error on parsing missing details file")
}
