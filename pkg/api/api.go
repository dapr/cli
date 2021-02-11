// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

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
