//go:build embed
// +build embed

package standalone

import "embed"

//go:embed staging
var binaries embed.FS

//go:embed staging/runtime.ver
var runtimeVersion string

//go:embed staging/dashboard.ver
var dashboardVersion string

var isEmbedded = true
