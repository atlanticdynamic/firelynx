package fileread

// Config contains everything needed to instantiate a fileread app.
// This is a Data Transfer Object (DTO) with no dependencies on domain packages.
type Config struct {
	// ID is the unique identifier for this app instance.
	ID string

	// BaseDirectory is the root directory for file read operations.
	BaseDirectory string
}
