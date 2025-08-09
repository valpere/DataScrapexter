//go:build !debug
// +build !debug

package proxy

// IsProductionBuild returns true if this is a production build
// This is determined at compile time using build tags
func IsProductionBuild() bool {
	return true
}