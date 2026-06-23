/*
Copyright 2026 The Dapr Authors
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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainersToRemove(t *testing.T) {
	t.Run("container mode uninstall removes sentry without warning", func(t *testing.T) {
		containers := containersToRemove(true, true, false)
		var sentry *removableContainer
		for i := range containers {
			if containers[i].name == DaprSentryContainerName {
				sentry = &containers[i]
				break
			}
		}
		require.NotNil(t, sentry)
		assert.False(t, sentry.warnIfMissing)
	})

	t.Run("uninstall all includes redis and zipkin", func(t *testing.T) {
		containers := containersToRemove(true, true, true)
		names := map[string]bool{}
		for _, c := range containers {
			names[c.name] = true
		}
		assert.True(t, names[DaprRedisContainerName])
		assert.True(t, names[DaprZipkinContainerName])
		assert.True(t, names[DaprSentryContainerName])
	})

	t.Run("sentry not removed if placement not removed and not uninstallAll", func(t *testing.T) {
		containers := containersToRemove(false, true, false)
		for _, c := range containers {
			assert.NotEqual(t, DaprSentryContainerName, c.name)
		}
	})
}

func TestRemoveDir(t *testing.T) {
	t.Run("remove existing directory", func(t *testing.T) {
		dir := t.TempDir()
		err := removeDir(dir)
		assert.NoError(t, err)
		_, err = os.Stat(dir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("remove non-existent directory", func(t *testing.T) {
		err := removeDir(filepath.Join(t.TempDir(), "non-existent"))
		assert.NoError(t, err)
	})
}
