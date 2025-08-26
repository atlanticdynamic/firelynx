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
	app := New(id)
	app.Response = proto.GetResponse()
	return app
}

// ToProto converts the EchoApp configuration to its protocol buffer representation
func (e *EchoApp) ToProto() any {
	return &pbApps.EchoApp{
		Response: &e.Response,
	}
}
