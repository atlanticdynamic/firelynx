// Package listeners provides domain model for network listeners
package listeners

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
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
		switch opts := l.Options.(type) {
		case options.HTTP:
			pbListener.ProtocolOptions = &pb.Listener_Http{
				Http: options.HTTPToProto(opts),
			}
		case options.GRPC:
			pbListener.ProtocolOptions = &pb.Listener_Grpc{
				Grpc: options.GRPCToProto(opts),
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
			listenerObj.Options = options.HTTPFromProto(http)
		} else if grpc := l.GetGrpc(); grpc != nil {
			listenerObj.Options = options.GRPCFromProto(grpc)
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
