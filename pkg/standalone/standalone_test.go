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
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandaloneConfig(t *testing.T) {
	t.Parallel()
	testFile := "./test.yaml"

	t.Run("Standalone config", func(t *testing.T) {
		t.Parallel()
		expectConfigZipkin := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec:
  tracing:
    samplingRate: "1"
    zipkin:
      endpointAddress: http://test_zipkin_host:9411/api/v2/spans
`
		os.Remove(testFile)
		createDefaultConfiguration("test_zipkin_host", testFile)
		assert.FileExists(t, testFile)
		content, err := ioutil.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, expectConfigZipkin, string(content))
	})

	t.Run("Standalone config slim", func(t *testing.T) {
		t.Parallel()
		expectConfigSlim := `apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: daprConfig
spec: {}
`
		os.Remove(testFile)
		createDefaultConfiguration("", testFile)
		assert.FileExists(t, testFile)
		content, err := ioutil.ReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, expectConfigSlim, string(content))
	})

	os.Remove(testFile)
}
