package echo

// Config contains everything needed to instantiate an echo app.
// This is a Data Transfer Object (DTO) with no dependencies on domain packages.
// All validation happens at the domain layer before creating this config.
type Config struct {
	// ID is the unique identifier for this app instance
	ID string

	// Response is the text content to return for HTTP requests
	Response string
}
