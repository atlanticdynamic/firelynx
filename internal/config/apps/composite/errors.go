package composite

import (
	"errors"
	"fmt"
)

var (
	// ErrAppCompositeScript is the base error type for AppCompositeScript errors.
	ErrAppCompositeScript = errors.New("app composite script error")

	// ErrNoScriptsSpecified indicates that no script IDs were provided.
	ErrNoScriptsSpecified = fmt.Errorf("%w: no scripts specified", ErrAppCompositeScript)

	// ErrEmptyScriptID indicates that an empty script ID was provided.
	ErrEmptyScriptID = fmt.Errorf("%w: empty script ID", ErrAppCompositeScript)

	// ErrInvalidStaticData indicates that the provided static data is invalid.
	ErrInvalidStaticData = fmt.Errorf("%w: invalid static data", ErrAppCompositeScript)

	// ErrProtoConversion indicates an error converting to/from protobuf.
	ErrProtoConversion = fmt.Errorf("%w: proto conversion error", ErrAppCompositeScript)
)
