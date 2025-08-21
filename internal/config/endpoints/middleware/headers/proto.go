package headers

import (
	"fmt"
	"maps"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
)

// ToProto converts Headers to protobuf format
func (h *Headers) ToProto() any {
	config := &pb.HeadersConfig{}

	// Convert request operations
	if h.Request != nil {
		config.Request = &pb.HeadersConfig_HeaderOperations{
			SetHeaders:    maps.Clone(h.Request.SetHeaders),
			AddHeaders:    maps.Clone(h.Request.AddHeaders),
			RemoveHeaders: make([]string, len(h.Request.RemoveHeaders)),
		}

		// Copy request remove headers
		copy(config.Request.RemoveHeaders, h.Request.RemoveHeaders)
	}

	// Convert response operations
	if h.Response != nil {
		config.Response = &pb.HeadersConfig_HeaderOperations{
			SetHeaders:    maps.Clone(h.Response.SetHeaders),
			AddHeaders:    maps.Clone(h.Response.AddHeaders),
			RemoveHeaders: make([]string, len(h.Response.RemoveHeaders)),
		}

		// Copy response remove headers
		copy(config.Response.RemoveHeaders, h.Response.RemoveHeaders)
	}

	return config
}

// newHeaderOperationsFromProto converts protobuf HeaderOperations to domain HeaderOperations
func newHeaderOperationsFromProto(title HeaderOperationsType, pbOps *pb.HeadersConfig_HeaderOperations) *HeaderOperations {
	if pbOps == nil {
		return nil
	}

	ops := &HeaderOperations{
		Title:         title,
		SetHeaders:    make(map[string]string),
		AddHeaders:    make(map[string]string),
		RemoveHeaders: make([]string, 0),
	}

	// Copy set headers if not nil
	if pbOps.SetHeaders != nil {
		ops.SetHeaders = maps.Clone(pbOps.SetHeaders)
	}

	// Copy add headers if not nil
	if pbOps.AddHeaders != nil {
		ops.AddHeaders = maps.Clone(pbOps.AddHeaders)
	}

	// Copy remove headers
	if len(pbOps.RemoveHeaders) > 0 {
		ops.RemoveHeaders = make([]string, len(pbOps.RemoveHeaders))
		copy(ops.RemoveHeaders, pbOps.RemoveHeaders)
	}

	return ops
}

// FromProto converts protobuf HeadersConfig to domain Headers
func FromProto(pbConfig *pb.HeadersConfig) (*Headers, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("nil headers config")
	}

	requestOps := newHeaderOperationsFromProto(
		RequestHeaderOperationsType, pbConfig.Request)

	responseOps := newHeaderOperationsFromProto(
		ResponseHeaderOperationsType, pbConfig.Response)

	return NewHeaders(requestOps, responseOps), nil
}
