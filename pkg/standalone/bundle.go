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
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const bundleDetailsFileName = "details.json"

type bundleDetails struct {
	RuntimeVersion    *string `json:"daprd"`
	DashboardVersion  *string `json:"dashboard"`
	CLIVersion        *string `json:"cli"`
	BinarySubDir      *string `json:"daprBinarySubDir"`
	ImageSubDir       *string `json:"dockerImageSubDir"`
	DaprImageName     *string `json:"daprImageName"`
	DaprImageFileName *string `json:"daprImageFileName"`
}

// readAndParseDetails reads the file in detailsFilePath and tries to parse it into the bundleDetails struct.
func (b *bundleDetails) readAndParseDetails(detailsFilePath string) error {
	bytes, err := os.ReadFile(detailsFilePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &b)
	if err != nil {
		return err
	}
	if isStringNilOrEmpty(b.RuntimeVersion) || isStringNilOrEmpty(b.DashboardVersion) ||
		isStringNilOrEmpty(b.DaprImageName) || isStringNilOrEmpty(b.DaprImageFileName) ||
		isStringNilOrEmpty(b.BinarySubDir) || isStringNilOrEmpty(b.ImageSubDir) {
		return fmt.Errorf("required fields are missing in %s", detailsFilePath)
	}
	return nil
}

func isStringNilOrEmpty(val *string) bool {
	return val == nil || strings.TrimSpace(*val) == ""
}

func (b *bundleDetails) getPlacementImageName() string {
	return *b.DaprImageName
}

func (b *bundleDetails) getPlacementImageFileName() string {
	return *b.DaprImageFileName
}
