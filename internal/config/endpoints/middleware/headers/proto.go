package headers

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
)

// ToProto converts Headers to protobuf format
func (h *Headers) ToProto() any {
	config := &pb.HeadersConfig{
		SetHeaders:    make(map[string]string),
		AddHeaders:    make(map[string]string),
		RemoveHeaders: make([]string, len(h.RemoveHeaders)),
	}

	// Copy set headers
	for key, value := range h.SetHeaders {
		config.SetHeaders[key] = value
	}

	// Copy add headers
	for key, value := range h.AddHeaders {
		config.AddHeaders[key] = value
	}

	// Copy remove headers
	copy(config.RemoveHeaders, h.RemoveHeaders)

	return config
}

// FromProto converts protobuf HeadersConfig to domain Headers
func FromProto(pbConfig *pb.HeadersConfig) (*Headers, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("nil headers config")
	}

	config := &Headers{
		SetHeaders:    make(map[string]string),
		AddHeaders:    make(map[string]string),
		RemoveHeaders: make([]string, 0),
	}

	// Copy set headers
	if pbConfig.SetHeaders != nil {
		for key, value := range pbConfig.SetHeaders {
			config.SetHeaders[key] = value
		}
	}

	// Copy add headers
	if pbConfig.AddHeaders != nil {
		for key, value := range pbConfig.AddHeaders {
			config.AddHeaders[key] = value
		}
	}

	// Copy remove headers
	if len(pbConfig.RemoveHeaders) > 0 {
		config.RemoveHeaders = make([]string, len(pbConfig.RemoveHeaders))
		copy(config.RemoveHeaders, pbConfig.RemoveHeaders)
	}

	return config, nil
}
