package config

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/styles"
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
	loggingTree := cfg.Logging.ToTree()
	t.Child(loggingTree.Tree())

	// Create a properly nested tree of listeners with consistent styling
	if len(cfg.Listeners) > 0 {
		listenersRoot := fancy.NewComponentTree(
			styles.FormatSection("Listeners", len(cfg.Listeners)),
		)
		for _, l := range cfg.Listeners {
			listenerTree := l.ToTree()
			listenersRoot.AddChild(listenerTree.Tree())
		}
		t.Child(listenersRoot.Tree())
	}

	// Create a properly nested tree of endpoints with consistent styling
	if len(cfg.Endpoints) > 0 {
		endpointsRoot := fancy.NewComponentTree(
			styles.FormatSection("Endpoints", len(cfg.Endpoints)),
		)
		for _, ep := range cfg.Endpoints {
			epTree := ep.ToTree()
			endpointsRoot.AddChild(epTree.Tree())
		}
		t.Child(endpointsRoot.Tree())
	}

	// Create a properly nested tree of apps with consistent styling
	if len(cfg.Apps) > 0 {
		appsRoot := fancy.NewComponentTree(styles.FormatSection("Apps", len(cfg.Apps)))
		for _, app := range cfg.Apps {
			appTree := app.ToTree()
			appsRoot.AddChild(appTree.Tree())
		}
		t.Child(appsRoot.Tree())
	}

	// Render the tree to string
	return t.String()
}
