// Package listeners provides domain model for network listeners
package listeners

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/robbyt/protobaggins"
)

// ToProto converts a Listeners collection to a slice of protobuf Listener messages
func (listeners ListenerCollection) ToProto() []*pb.Listener {
	if len(listeners) == 0 {
		return nil
	}

	pbListeners := make([]*pb.Listener, 0, len(listeners))
	for _, l := range listeners {
		pbListener := &pb.Listener{
			Id:      protobaggins.StringToProto(l.ID),
			Address: protobaggins.StringToProto(l.Address),
		}

		// Convert the listener type
		pbType := pb.Listener_Type(l.Type)
		pbListener.Type = &pbType

		// Convert options
		switch opts := l.Options.(type) {
		case options.HTTP:
			pbListener.ProtocolOptions = &pb.Listener_Http{
				Http: options.HTTPToProto(opts),
			}
		}

		pbListeners = append(pbListeners, pbListener)
	}

	return pbListeners
}

// FromProto converts protobuf Listener messages to a domain Listeners collection
func FromProto(pbListeners []*pb.Listener) (ListenerCollection, error) {
	if len(pbListeners) == 0 {
		return nil, nil
	}

	listeners := make(ListenerCollection, 0, len(pbListeners))
	for _, l := range pbListeners {
		listenerObj := Listener{
			ID:      protobaggins.StringFromProto(l.Id),
			Address: protobaggins.StringFromProto(l.Address),
		}

		// Convert the type field
		if l.Type != nil {
			listenerObj.Type = Type(*l.Type)
		}

		// Convert protocol-specific options
		if http := l.GetHttp(); http != nil {
			listenerObj.Options = options.HTTPFromProto(http)
		} else {
			return nil, fmt.Errorf("listener '%s' has unknown protocol options", listenerObj.ID)
		}

		listeners = append(listeners, listenerObj)
	}

	return listeners, nil
}
