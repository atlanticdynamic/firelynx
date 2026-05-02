package calculation

import pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"

// FromProto creates an App configuration from its protocol buffer representation.
func FromProto(id string, proto *pbApps.CalculationApp) *App {
	if proto == nil {
		return nil
	}
	return New(id)
}

// ToProto converts the App configuration to its protocol buffer representation.
func (a *App) ToProto() any {
	return &pbApps.CalculationApp{}
}
