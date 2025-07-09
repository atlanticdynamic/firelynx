// Package config provides domain model for server configuration
package config

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
)

// ToProto converts the domain Config to a protobuf ServerConfig
func (c *Config) ToProto() *pb.ServerConfig {
	config := &pb.ServerConfig{
		Version: &c.Version,
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

	// Convert listeners using the listeners package's FromProto method
	listeners, err := listeners.FromProto(pbConfig.Listeners)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
	}
	config.Listeners = listeners

	// Convert endpoints using endpoints package's FromProto method
	if len(pbConfig.Endpoints) > 0 {
		endpointsList, err := endpoints.FromProto(pbConfig.Endpoints)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
		}
		config.Endpoints = endpointsList
	}

	// Convert apps using the apps package's FromProto method
	if len(pbConfig.Apps) > 0 {
		appDefinitions, err := apps.FromProto(pbConfig.Apps)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
		}
		config.Apps = appDefinitions
		// Assign app instances to routes with merged static data
		expandAppsForRoutes(config.Apps, config.Endpoints)
	}

	return config, nil
}
