package composite

import (
	"fmt"

	settingsv1alpha1 "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// FromProto creates a CompositeScript from its protocol buffer representation.
func FromProto(proto *settingsv1alpha1.CompositeScriptApp) (*CompositeScript, error) {
	if proto == nil {
		return nil, nil
	}

	// Parse the static data
	sd, err := staticdata.FromProto(proto.StaticData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProtoConversion, err)
	}

	// Create and return the CompositeScript
	return &CompositeScript{
		ScriptAppIDs: proto.ScriptAppIds,
		StaticData:   sd,
	}, nil
}

// ToProto converts a CompositeScript to its protocol buffer representation.
func (s *CompositeScript) ToProto() any {
	if s == nil {
		return nil
	}

	// Create the protobuf message
	proto := &settingsv1alpha1.CompositeScriptApp{
		ScriptAppIds: s.ScriptAppIDs,
	}

	// Convert static data if present
	if s.StaticData != nil {
		proto.StaticData = s.StaticData.ToProto()
	}

	return proto
}
