package fileread

import pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"

// FromProto creates an App configuration from its protocol buffer representation.
func FromProto(id string, proto *pbApps.FileReadApp) *App {
	if proto == nil {
		return nil
	}
	app := New(id)
	app.BaseDirectory = proto.GetBaseDirectory()
	return app
}

// ToProto converts the App configuration to its protocol buffer representation.
func (a *App) ToProto() any {
	return &pbApps.FileReadApp{
		BaseDirectory: &a.BaseDirectory,
	}
}
