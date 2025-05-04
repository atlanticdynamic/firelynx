package options

import (
	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// HTTPFromProto converts protobuf HttpListenerOptions to domain HTTPOptions
func HTTPFromProto(pbOpts *pb.HttpListenerOptions) HTTP {
	opts := NewHTTP()
	if pbOpts == nil {
		return opts
	}

	if pbOpts.ReadTimeout != nil {
		d := pbOpts.ReadTimeout.AsDuration()
		if d > 0 {
			opts.ReadTimeout = d
		}
	}

	if pbOpts.WriteTimeout != nil {
		d := pbOpts.WriteTimeout.AsDuration()
		if d > 0 {
			opts.WriteTimeout = d
		}
	}

	if pbOpts.DrainTimeout != nil {
		d := pbOpts.DrainTimeout.AsDuration()
		if d > 0 {
			opts.DrainTimeout = d
		}
	}

	if pbOpts.IdleTimeout != nil {
		d := pbOpts.IdleTimeout.AsDuration()
		if d > 0 {
			opts.IdleTimeout = d
		}
	}

	return opts
}

// HTTPToProto converts domain HTTPOptions to protobuf HttpListenerOptions
func HTTPToProto(opts HTTP) *pb.HttpListenerOptions {
	pbOpts := &pb.HttpListenerOptions{
		ReadTimeout:  durationpb.New(opts.ReadTimeout),
		WriteTimeout: durationpb.New(opts.WriteTimeout),
		DrainTimeout: durationpb.New(opts.DrainTimeout),
		IdleTimeout:  durationpb.New(opts.IdleTimeout),
	}
	return pbOpts
}

// GRPCFromProto converts protobuf GrpcListenerOptions to domain GRPCOptions
func GRPCFromProto(pbOpts *pb.GrpcListenerOptions) GRPC {
	opts := NewGRPC()
	if pbOpts == nil {
		return opts
	}

	// Convert each field if present
	if pbOpts.MaxConnectionIdle != nil {
		d := pbOpts.MaxConnectionIdle.AsDuration()
		if d > 0 {
			opts.MaxConnectionIdle = d
		}
	}

	if pbOpts.MaxConnectionAge != nil {
		d := pbOpts.MaxConnectionAge.AsDuration()
		if d > 0 {
			opts.MaxConnectionAge = d
		}
	}

	if pbOpts.MaxConcurrentStreams != nil {
		streams := int(pbOpts.GetMaxConcurrentStreams())
		if streams >= 0 {
			opts.MaxConcurrentStreams = streams
		}
	}

	return opts
}

// GRPCToProto converts domain GRPCOptions to protobuf GrpcListenerOptions
func GRPCToProto(opts GRPC) *pb.GrpcListenerOptions {
	maxStreams := int32(opts.MaxConcurrentStreams)
	pbOpts := &pb.GrpcListenerOptions{
		MaxConnectionIdle:    durationpb.New(opts.MaxConnectionIdle),
		MaxConnectionAge:     durationpb.New(opts.MaxConnectionAge),
		MaxConcurrentStreams: &maxStreams,
	}
	return pbOpts
}
