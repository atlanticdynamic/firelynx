package config

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
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
		appNode := appsTree.Child(fancy.AppText(app.ID))

		// Determine app type and add specific details
		switch appConfig := app.Config.(type) {
		case apps.ScriptApp:
			appNode.Child("Type: Script")

			// Add evaluator type and info
			switch eval := appConfig.Evaluator.(type) {
			case apps.RisorEvaluator:
				evalNode := appNode.Child("Evaluator: Risor")
				codePreview := fancy.TruncateString(eval.Code, 40)
				evalNode.Child(fmt.Sprintf("Code: %s", codePreview))
				if eval.Timeout != nil {
					evalNode.Child(
						fmt.Sprintf("Timeout: %v", eval.Timeout.AsDuration()),
					)
				}
			case apps.StarlarkEvaluator:
				evalNode := appNode.Child("Evaluator: Starlark")
				codePreview := fancy.TruncateString(eval.Code, 40)
				evalNode.Child(fmt.Sprintf("Code: %s", codePreview))
				if eval.Timeout != nil {
					evalNode.Child(
						fmt.Sprintf("Timeout: %v", eval.Timeout.AsDuration()),
					)
				}
			case apps.ExtismEvaluator:
				evalNode := appNode.Child("Evaluator: Extism")
				evalNode.Child(fmt.Sprintf("Entrypoint: %s", eval.Entrypoint))
				codePreview := fmt.Sprintf("<%d bytes>", len(eval.Code))
				evalNode.Child(fmt.Sprintf("Code: %s", codePreview))
			}

			// Add static data if present
			if len(appConfig.StaticData.Data) > 0 {
				dataNode := appNode.Child("StaticData")
				dataNode.Child(
					fmt.Sprintf("MergeMode: %s", appConfig.StaticData.MergeMode),
				)
				for k, v := range appConfig.StaticData.Data {
					dataNode.Child(fmt.Sprintf("%s: %v", k, v))
				}
			}
		case apps.CompositeScriptApp:
			appNode.Child("Type: CompositeScript")

			// Add script apps
			if len(appConfig.ScriptAppIDs) > 0 {
				scriptsNode := appNode.Child("ScriptApps")
				for _, scriptID := range appConfig.ScriptAppIDs {
					scriptsNode.Child(scriptID)
				}
			}

			// Add static data if present
			if len(appConfig.StaticData.Data) > 0 {
				dataNode := appNode.Child("StaticData")
				dataNode.Child(
					fmt.Sprintf("MergeMode: %s", appConfig.StaticData.MergeMode),
				)
				for k, v := range appConfig.StaticData.Data {
					dataNode.Child(fmt.Sprintf("%s: %v", k, v))
				}
			}
		}
	}

	// Render the tree to string
	return t.String()
}
