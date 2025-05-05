package composite

import (
	"fmt"

	settingsv1alpha1 "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// FromProto creates an AppCompositeScript from its protocol buffer representation.
func FromProto(proto *settingsv1alpha1.AppCompositeScript) (*AppCompositeScript, error) {
	if proto == nil {
		return nil, nil
	}

	// Parse the static data
	staticData, err := staticdata.FromProto(proto.StaticData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProtoConversion, err)
	}

	// Create and return the AppCompositeScript
	return &AppCompositeScript{
		ScriptAppIDs: proto.ScriptAppIds,
		StaticData:   staticData,
	}, nil
}

// ToProto converts an AppCompositeScript to its protocol buffer representation.
func (s *AppCompositeScript) ToProto() (*settingsv1alpha1.AppCompositeScript, error) {
	if s == nil {
		return nil, nil
	}

	// Create the protobuf message
	proto := &settingsv1alpha1.AppCompositeScript{
		ScriptAppIds: s.ScriptAppIDs,
	}

	// Convert static data if present
	if s.StaticData != nil {
		staticDataProto, err := s.StaticData.ToProto()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrProtoConversion, err)
		}
		proto.StaticData = staticDataProto
	}

	return proto, nil
}
