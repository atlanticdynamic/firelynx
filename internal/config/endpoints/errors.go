package endpoints

// Import errors from the centralized errz package
import (
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// Re-export errors for this package
var (
	// Validation specific errors
	ErrEmptyID              = errz.ErrEmptyID
	ErrMissingRequiredField = errz.ErrMissingRequiredField
	ErrRouteConflict        = errz.ErrRouteConflict
	ErrInvalidRouteType     = errz.ErrInvalidRouteType
)
