// Package echo provides app-specific configurations for the firelynx server.
//
// This file contains functions for converting between domain models and protocol buffers.
package echo

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// EchoFromProto creates an EchoApp configuration from its protocol buffer representation
func EchoFromProto(proto *pb.EchoApp) *EchoApp {
	if proto == nil {
		return nil
	}
	return &EchoApp{
		Response: proto.GetResponse(),
	}
}

// ToProto converts the EchoApp configuration to its protocol buffer representation
func (e *EchoApp) ToProto() any {
	return &pb.EchoApp{
		Response: &e.Response,
	}
}
