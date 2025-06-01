package middleware

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
)

// ToProto converts a MiddlewareCollection to protobuf format
func (mc MiddlewareCollection) ToProto() []*pb.Middleware {
	if len(mc) == 0 {
		return nil
	}

	result := make([]*pb.Middleware, len(mc))
	for i, middleware := range mc {
		result[i] = middleware.ToProto()
	}
	return result
}

// ToProto converts a single Middleware to protobuf format
func (m Middleware) ToProto() *pb.Middleware {
	pbMiddleware := &pb.Middleware{
		Id: &m.ID,
	}

	switch config := m.Config.(type) {
	case *logger.ConsoleLogger:
		pbMiddleware.Type = pb.Middleware_TYPE_CONSOLE_LOGGER.Enum()
		pbMiddleware.Config = &pb.Middleware_ConsoleLogger{
			ConsoleLogger: config.ToProto().(*pb.ConsoleLoggerConfig),
		}
	default:
		// Unknown middleware type - this should be caught during validation
		pbMiddleware.Type = pb.Middleware_TYPE_UNSPECIFIED.Enum()
	}

	return pbMiddleware
}

// FromProto converts protobuf middlewares to domain MiddlewareCollection
func FromProto(pbMiddlewares []*pb.Middleware) (MiddlewareCollection, error) {
	if len(pbMiddlewares) == 0 {
		return nil, nil
	}

	middlewares := make(MiddlewareCollection, len(pbMiddlewares))
	for i, pbMiddleware := range pbMiddlewares {
		middleware, err := middlewareFromProto(pbMiddleware)
		if err != nil {
			return nil, fmt.Errorf("middleware at index %d: %w", i, err)
		}
		middlewares[i] = middleware
	}

	return middlewares, nil
}

// middlewareFromProto converts a single protobuf Middleware to domain Middleware
func middlewareFromProto(pbMiddleware *pb.Middleware) (Middleware, error) {
	if pbMiddleware.GetId() == "" {
		return Middleware{}, fmt.Errorf("middleware has empty ID")
	}

	middleware := Middleware{
		ID: pbMiddleware.GetId(),
	}

	switch pbMiddleware.GetType() {
	case pb.Middleware_TYPE_CONSOLE_LOGGER:
		if consoleConfig := pbMiddleware.GetConsoleLogger(); consoleConfig != nil {
			config, err := logger.FromProto(consoleConfig)
			if err != nil {
				return Middleware{}, fmt.Errorf("console logger config: %w", err)
			}
			middleware.Config = config
		} else {
			return Middleware{}, fmt.Errorf("console logger middleware missing config")
		}
	case pb.Middleware_TYPE_UNSPECIFIED:
		return Middleware{}, fmt.Errorf("middleware type unspecified")
	default:
		return Middleware{}, fmt.Errorf("unknown middleware type: %v", pbMiddleware.GetType())
	}

	return middleware, nil
}
