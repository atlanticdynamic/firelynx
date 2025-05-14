package staticdata

// StaticDataMergeMode represents the strategy for merging static data from different sources.
type StaticDataMergeMode int

// StaticDataMergeMode enum values - must match the protobuf definition.
const (
	// StaticDataMergeModeUnspecified is the default merge mode, where behavior might be defined by the consuming system.
	StaticDataMergeModeUnspecified StaticDataMergeMode = iota

	// StaticDataMergeModeLast uses the last value found (highest priority source wins).
	// Later static_data completely replaces earlier ones.
	StaticDataMergeModeLast

	// StaticDataMergeModeUnique ensures that if a key exists in multiple sources,
	// the values from the last key will replace earlier keys.
	StaticDataMergeModeUnique
)

// StaticData represents a collection of static data with a merge strategy.
// Static data can be passed to apps and routes to provide configuration values.
type StaticData struct {
	// Data contains the key-value pairs of the static data.
	Data map[string]any

	// MergeMode defines how this static data should be merged with other static data when combined.
	MergeMode StaticDataMergeMode
}
