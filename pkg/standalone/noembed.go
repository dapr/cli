//go:build !embed
// +build !embed

package standalone

import "embed"

var binaries embed.FS
var runtimeVersion string
var dashboardVersion string
var isEmbedded = false
