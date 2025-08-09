//go:build debug
// +build debug

package proxy

// IsProductionBuild returns false for debug builds
// This allows more lenient security settings during development
func IsProductionBuild() bool {
	return false
}