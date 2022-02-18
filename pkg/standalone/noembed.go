//go:build !embed
// +build !embed

package standalone

import "embed"

var (
	binaries         embed.FS
	runtimeVersion   string
	dashboardVersion string
	isEmbedded       = false
)
