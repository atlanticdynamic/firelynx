package config

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a pretty-printed tree representation of the config
func (c *Config) String() string {
	return ConfigTree(c)
}

// ConfigTree converts a Config struct into a rendered tree string
func ConfigTree(cfg *Config) string {
	// Set up the root node with the config version
	t := fancy.Tree()
	t.Root(fancy.RootStyle.Render(fmt.Sprintf("Firelynx Config (%s)", cfg.Version)))

	// Add logging section
	loggingTree := t.Child("Logging")
	loggingTree.Child(fmt.Sprintf("Format: %s", cfg.Logging.Format))
	loggingTree.Child(fmt.Sprintf("Level: %s", cfg.Logging.Level))

	// Add listeners section
	listenersTree := t.Child("Listeners")
	for _, l := range cfg.Listeners {
		// Use the listener's ToTree method to get its tree representation
		listenerTree := l.ToTree()
		listenersTree.Child(listenerTree)
	}

	// Add endpoints section
	endpointsTree := t.Child("Endpoints")
	for _, ep := range cfg.Endpoints {
		// Use the endpoint's ToTree method to get its tree representation
		epTree := ep.ToTree()
		endpointsTree.Child(epTree)
	}

	// Add apps section
	appsTree := t.Child("Apps")
	for _, app := range cfg.Apps {
		// Use the app's ToTree method to get its tree representation
		appTree := app.ToTree()
		appsTree.Child(appTree)
	}

	// Render the tree to string
	return t.String()
}
