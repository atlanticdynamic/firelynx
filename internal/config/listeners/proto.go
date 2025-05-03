// Package listeners provides domain model for network listeners
package listeners

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// ToProto converts a Listeners collection to a slice of protobuf Listener messages
func (listeners Listeners) ToProto() []*pb.Listener {
	if len(listeners) == 0 {
		return nil
	}

	pbListeners := make([]*pb.Listener, 0, len(listeners))
	for _, l := range listeners {
		pbListener := &pb.Listener{
			Id:      &l.ID,
			Address: &l.Address,
		}

		// Convert options
		if httpOpts, ok := l.Options.(HTTPOptions); ok {
			pbListener.ProtocolOptions = &pb.Listener_Http{
				Http: &pb.HttpListenerOptions{
					ReadTimeout:  httpOpts.ReadTimeout,
					WriteTimeout: httpOpts.WriteTimeout,
					IdleTimeout:  httpOpts.IdleTimeout,
					DrainTimeout: httpOpts.DrainTimeout,
				},
			}
		} else if grpcOpts, ok := l.Options.(GRPCOptions); ok {
			maxStreams := int32(grpcOpts.MaxConcurrentStreams)
			pbListener.ProtocolOptions = &pb.Listener_Grpc{
				Grpc: &pb.GrpcListenerOptions{
					MaxConnectionIdle:    grpcOpts.MaxConnectionIdle,
					MaxConnectionAge:     grpcOpts.MaxConnectionAge,
					MaxConcurrentStreams: &maxStreams,
				},
			}
		}

		pbListeners = append(pbListeners, pbListener)
	}

	return pbListeners
}

// FromProto converts protobuf Listener messages to a domain Listeners collection
func FromProto(pbListeners []*pb.Listener) (Listeners, error) {
	if len(pbListeners) == 0 {
		return nil, nil
	}

	listeners := make(Listeners, 0, len(pbListeners))
	for _, l := range pbListeners {
		listenerObj := Listener{
			ID:      getStringValue(l.Id),
			Address: getStringValue(l.Address),
		}

		// Convert protocol-specific options
		if http := l.GetHttp(); http != nil {
			listenerObj.Type = TypeHTTP
			listenerObj.Options = HTTPOptions{
				ReadTimeout:  http.ReadTimeout,
				WriteTimeout: http.WriteTimeout,
				DrainTimeout: http.DrainTimeout,
				IdleTimeout:  http.IdleTimeout,
			}
		} else if grpc := l.GetGrpc(); grpc != nil {
			listenerObj.Type = TypeGRPC
			listenerObj.Options = GRPCOptions{
				MaxConnectionIdle:    grpc.MaxConnectionIdle,
				MaxConnectionAge:     grpc.MaxConnectionAge,
				MaxConcurrentStreams: int(grpc.GetMaxConcurrentStreams()),
			}
		} else {
			return nil, fmt.Errorf("listener '%s' has unknown protocol options", listenerObj.ID)
		}

		listeners = append(listeners, listenerObj)
	}

	return listeners, nil
}

// Helper function to safely get string value from a string pointer
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
