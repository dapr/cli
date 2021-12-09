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

package api

// RuntimeAPIVersion represents the version for the Dapr runtime API.
var (
	RuntimeAPIVersion = "1.0"
)

// Metadata representa information about sidecar.
type Metadata struct {
	ID                string                      `json:"id"`
	ActiveActorsCount []MetadataActiveActorsCount `json:"actors"`
	Extended          map[string]string           `json:"extended"`
}

// MetadataActiveActorsCount contain actorType and count of actors each type has.
type MetadataActiveActorsCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}
