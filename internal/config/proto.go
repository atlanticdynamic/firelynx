// Package config provides domain model for server configuration
package config

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/logs"
)

// ToProto converts the domain Config to a protobuf ServerConfig
func (c *Config) ToProto() *pb.ServerConfig {
	config := &pb.ServerConfig{
		Version: &c.Version,
	}

	if c.Logging.Format != "" || c.Logging.Level != "" {
		config.Logging = c.Logging.ToProto()
	}

	config.Listeners = c.Listeners.ToProto()
	config.Endpoints = c.Endpoints.ToProto()
	config.Apps = c.Apps.ToProto()

	return config
}

// fromProto is an internal function that performs basic conversion from protobuf to domain model.
// For public use, prefer NewFromProto which handles defaults and additional initialization.
func fromProto(pbConfig *pb.ServerConfig) (*Config, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("%w: nil protobuf config", ErrFailedToConvertConfig)
	}

	if pbConfig.Version == nil {
		return nil, fmt.Errorf("%w: nil version", ErrFailedToConvertConfig)
	}

	config := &Config{
		Version:  *pbConfig.Version,
		rawProto: pbConfig,
	}

	// Convert logging config
	if pbConfig.Logging != nil {
		config.Logging = logs.FromProto(pbConfig.Logging)
	}

	// Convert listeners using the listeners package's FromProto method
	listeners, err := listeners.FromProto(pbConfig.Listeners)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
	}
	config.Listeners = listeners

	// Convert endpoints using endpoints package's FromProto method
	if len(pbConfig.Endpoints) > 0 {
		config.Endpoints = endpoints.NewEndpointsFromProto(pbConfig.Endpoints...)
	}

	// Convert apps using the apps package's FromProto method
	if len(pbConfig.Apps) > 0 {
		appDefinitions, err := apps.FromProto(pbConfig.Apps)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
		}
		config.Apps = appDefinitions
	}

	return config, nil
}
