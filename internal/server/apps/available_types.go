// Package apps provides interfaces and implementations for firelynx applications.
package apps

// GetAvailableAppTypes returns a list of all app types that are implemented in the codebase.
// This is used for validation in the config layer.
func GetAvailableAppTypes() []string {
	types := make([]string, 0, len(AvailableAppImplementations))
	for appType := range AvailableAppImplementations {
		types = append(types, appType)
	}
	return types
}

// GetBuiltInAppIDs returns a list of app IDs that are always available.
// These are distinct from app types - they are specific instances that
// are always registered in the runtime.
func GetBuiltInAppIDs() []string {
	// Currently, only the "echo" app is built-in
	return []string{"echo"}
}
