package options

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// HTTP default timeout values
const (
	DefaultHTTPReadTimeout  = 10 * time.Second
	DefaultHTTPWriteTimeout = 10 * time.Second
	DefaultHTTPDrainTimeout = 30 * time.Second
	DefaultHTTPIdleTimeout  = 60 * time.Second
)

// HTTP contains HTTP-specific listener configuration
type HTTP struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	DrainTimeout time.Duration
	IdleTimeout  time.Duration
}

// NewHTTP creates a new HTTP with default values
func NewHTTP() HTTP {
	return HTTP{
		ReadTimeout:  DefaultHTTPReadTimeout,
		WriteTimeout: DefaultHTTPWriteTimeout,
		DrainTimeout: DefaultHTTPDrainTimeout,
		IdleTimeout:  DefaultHTTPIdleTimeout,
	}
}

// Type returns the listener type this options is for
func (h HTTP) Type() Type { return TypeHTTP }

// Validate checks HTTP for any configuration errors
func (h HTTP) Validate() error {
	var errs []error

	// Validate timeouts (should be positive)
	if h.ReadTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP read timeout must be positive",
			errz.ErrInvalidValue))
	}

	if h.WriteTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP write timeout must be positive",
			errz.ErrInvalidValue))
	}

	if h.DrainTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP drain timeout must be positive",
			errz.ErrInvalidValue))
	}

	if h.IdleTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%w: HTTP idle timeout must be positive",
			errz.ErrInvalidValue))
	}

	return errors.Join(errs...)
}

// GetReadTimeout returns the read timeout
func (h HTTP) GetReadTimeout() time.Duration {
	if h.ReadTimeout <= 0 {
		return DefaultHTTPReadTimeout
	}
	return h.ReadTimeout
}

// GetWriteTimeout returns the write timeout
func (h HTTP) GetWriteTimeout() time.Duration {
	if h.WriteTimeout <= 0 {
		return DefaultHTTPWriteTimeout
	}
	return h.WriteTimeout
}

// GetDrainTimeout returns the drain timeout
func (h HTTP) GetDrainTimeout() time.Duration {
	if h.DrainTimeout <= 0 {
		return DefaultHTTPDrainTimeout
	}
	return h.DrainTimeout
}

// GetIdleTimeout returns the idle timeout
func (h HTTP) GetIdleTimeout() time.Duration {
	if h.IdleTimeout <= 0 {
		return DefaultHTTPIdleTimeout
	}
	return h.IdleTimeout
}

// String returns a concise string representation of HTTP options
func (h HTTP) String() string {
	var b strings.Builder
	if h.ReadTimeout > 0 {
		fmt.Fprintf(&b, "ReadTimeout: %v, ", h.ReadTimeout)
	}
	if h.WriteTimeout > 0 {
		fmt.Fprintf(&b, "WriteTimeout: %v, ", h.WriteTimeout)
	}
	if h.IdleTimeout > 0 {
		fmt.Fprintf(&b, "IdleTimeout: %v, ", h.IdleTimeout)
	}
	if h.DrainTimeout > 0 {
		fmt.Fprintf(&b, "DrainTimeout: %v, ", h.DrainTimeout)
	}

	str := b.String()
	if len(str) > 2 {
		// Remove trailing comma and space
		return str[:len(str)-2]
	}
	return str
}

// ToTree returns a tree visualization of HTTP options
func (h HTTP) ToTree() *fancy.ComponentTree {
	// Create a base tree for the HTTP options
	tree := fancy.NewComponentTree("HTTP Options")

	if h.ReadTimeout > 0 {
		tree.AddChild(fmt.Sprintf("ReadTimeout: %v", h.ReadTimeout))
	}
	if h.WriteTimeout > 0 {
		tree.AddChild(fmt.Sprintf("WriteTimeout: %v", h.WriteTimeout))
	}
	if h.IdleTimeout > 0 {
		tree.AddChild(fmt.Sprintf("IdleTimeout: %v", h.IdleTimeout))
	}
	if h.DrainTimeout > 0 {
		tree.AddChild(fmt.Sprintf("DrainTimeout: %v", h.DrainTimeout))
	}

	return tree
}
