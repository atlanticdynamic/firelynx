// Package echo provides app-specific configurations for the firelynx server.
//
// This file contains functions for converting between domain models and protocol buffers.
package echo

import (
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
)

// EchoFromProto creates an EchoApp configuration from its protocol buffer representation
func EchoFromProto(id string, proto *pbApps.EchoApp) *EchoApp {
	if proto == nil {
		return nil
	}
	// Note: ID will be set by the calling code from AppDefinition
	return New(id, proto.GetResponse())
}

// ToProto converts the EchoApp configuration to its protocol buffer representation
func (e *EchoApp) ToProto() any {
	return &pbApps.EchoApp{
		Response: &e.Response,
	}
}
