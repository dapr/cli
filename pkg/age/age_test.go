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

package age

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetAge(t *testing.T) {
	now := time.Now()

	t.Run("seconds", func(t *testing.T) {
		assert.Equal(t, "0s", GetAge(now))
		assert.Equal(t, "30s", GetAge(now.Add(-30*time.Second)))
		assert.Equal(t, "59s", GetAge(now.Add(-59*time.Second)))
	})

	t.Run("minutes", func(t *testing.T) {
		assert.Equal(t, "1m", GetAge(now.Add(-61*time.Second)))
		assert.Equal(t, "30m", GetAge(now.Add(-30*time.Minute)))
		assert.Equal(t, "59m", GetAge(now.Add(-59*time.Minute)))
	})

	t.Run("hours", func(t *testing.T) {
		assert.Equal(t, "1h", GetAge(now.Add(-61*time.Minute)))
		assert.Equal(t, "12h", GetAge(now.Add(-12*time.Hour)))
		assert.Equal(t, "23h", GetAge(now.Add(-23*time.Hour)))
	})

	t.Run("days", func(t *testing.T) {
		assert.Equal(t, "1d", GetAge(now.Add(-25*time.Hour)))
		assert.Equal(t, "7d", GetAge(now.Add(-7*24*time.Hour)))
		assert.Equal(t, "365d", GetAge(now.Add(-365*24*time.Hour)))
	})
}
