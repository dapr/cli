package kubernetes

// This can be removed in the future (>= 1.0) when chart versions align to runtime versions.
var chartVersionsMap = map[string]string{
	"0.7.0": "0.4.0",
	"0.7.1": "0.4.1",
	"0.8.0": "0.4.2",
	"0.9.0": "0.4.3",
}

// chartVersion will return the corresponding Helm Chart version for the given runtime version.
// If the specified version is not found, it is assumed that the chart version equals the runtime version.
func chartVersion(runtimeVersion string) string {
	v, ok := chartVersionsMap[runtimeVersion]
	if ok {
		return v
	}
	return runtimeVersion
}
