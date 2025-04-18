package config

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// String returns a pretty-printed tree representation of the config
func (c *Config) String() string {
	return ConfigTree(c)
}

// Config-specific styles
var (
	listenerStyle = lipgloss.NewStyle().Foreground(fancy.ColorMagenta)
	endpointStyle = lipgloss.NewStyle().Foreground(fancy.ColorOrange)
	appStyle      = lipgloss.NewStyle().Foreground(fancy.ColorGreen)
	routeStyle    = lipgloss.NewStyle().Foreground(fancy.ColorYellow)
)

// ConfigTree converts a Config struct into a rendered tree string
func ConfigTree(cfg *Config) string {
	// Set up the root node with the config version
	t := fancy.Tree()
	root := t.Root(fancy.RootStyle.Render(fmt.Sprintf("Firelynx Config (%s)", cfg.Version)))

	// Add Listeners section
	if len(cfg.Listeners) > 0 {
		listenersNode := fancy.BranchNode("Listeners", fmt.Sprintf("(%d)", len(cfg.Listeners)))

		for _, l := range cfg.Listeners {
			typeInfo := string(l.Type)
			listenerNode := tree.New().Root(listenerStyle.Render(l.ID))

			// Add address and type
			listenerNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Address: %s", l.Address)))
			listenerNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Type: %s", typeInfo)))

			// Add specific options based on listener type
			if l.Options != nil {
				optionsNode := tree.New().Root(fancy.ComponentStyle.Render("Options"))
				switch opts := l.Options.(type) {
				case HTTPListenerOptions:
					if opts.ReadTimeout != nil {
						optionsNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("ReadTimeout: %v", opts.ReadTimeout)))
					}
					if opts.WriteTimeout != nil {
						optionsNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("WriteTimeout: %v", opts.WriteTimeout)))
					}
					if opts.DrainTimeout != nil {
						optionsNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("DrainTimeout: %v", opts.DrainTimeout)))
					}
				case GRPCListenerOptions:
					if opts.MaxConnectionIdle != nil {
						optionsNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("MaxConnectionIdle: %v", opts.MaxConnectionIdle)))
					}
					if opts.MaxConnectionAge != nil {
						optionsNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("MaxConnectionAge: %v", opts.MaxConnectionAge)))
					}
					if opts.MaxConcurrentStreams > 0 {
						optionsNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("MaxConcurrentStreams: %d", opts.MaxConcurrentStreams)))
					}
				}

				// Only add options node if it has children
				if len(optionsNode.String()) > 0 {
					listenerNode.Child(optionsNode)
				}
			}

			listenersNode.Child(listenerNode)
		}

		root.Child(listenersNode)
	}

	// Add Endpoints section
	if len(cfg.Endpoints) > 0 {
		endpointsNode := fancy.BranchNode("Endpoints", fmt.Sprintf("(%d)", len(cfg.Endpoints)))

		for _, e := range cfg.Endpoints {
			endpointNode := tree.New().Root(endpointStyle.Render(e.ID))

			// Add listener references
			if len(e.ListenerIDs) > 0 {
				listenerPrefix := fancy.InfoStyle.Render("Listeners: ")
				listenerValue := listenerStyle.Render(strings.Join(e.ListenerIDs, ", "))
				listenersRef := lipgloss.JoinHorizontal(lipgloss.Top, listenerPrefix, listenerValue)
				endpointNode.Child(listenersRef)
			}

			// Add routes
			if len(e.Routes) > 0 {
				routesNode := tree.New().
					Root(fancy.ComponentStyle.Render(fmt.Sprintf("Routes (%d)", len(e.Routes))))

				for i, r := range e.Routes {
					routeNode := tree.New().Root(routeStyle.Render(fmt.Sprintf("Route %d", i+1)))

					// Add app reference
					appPrefix := fancy.InfoStyle.Render("App: ")
					appValue := appStyle.Render(r.AppID)
					routeNode.Child(lipgloss.JoinHorizontal(lipgloss.Top, appPrefix, appValue))

					// Add condition
					if r.Condition != nil {
						condType := r.Condition.Type()
						condValue := r.Condition.Value()
						routeNode.Child(
							fancy.InfoStyle.Render(
								fmt.Sprintf("Condition: %s = %s", condType, condValue),
							),
						)
					}

					// Add static data if present
					if len(r.StaticData) > 0 {
						staticNode := tree.New().Root(fancy.ComponentStyle.Render("Static Data"))
						for k, v := range r.StaticData {
							staticNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("%s: %v", k, v)))
						}
						routeNode.Child(staticNode)
					}

					routesNode.Child(routeNode)
				}

				endpointNode.Child(routesNode)
			}

			endpointsNode.Child(endpointNode)
		}

		root.Child(endpointsNode)
	}

	// Add Apps section
	if len(cfg.Apps) > 0 {
		appsNode := fancy.BranchNode("Apps", fmt.Sprintf("(%d)", len(cfg.Apps)))

		for _, a := range cfg.Apps {
			appNode := tree.New().Root(appStyle.Render(a.ID))

			// Add app type and specific configurations
			switch appConfig := a.Config.(type) {
			case ScriptApp:
				appNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Type: Script (%s)", appConfig.Evaluator.Type())))

				// Add evaluator details based on type
				switch eval := appConfig.Evaluator.(type) {
				case RisorEvaluator:
					evalNode := tree.New().Root(fancy.ComponentStyle.Render("Risor Evaluator"))
					if eval.Timeout != nil {
						evalNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Timeout: %v", eval.Timeout)))
					}

					codePreview := truncateCode(eval.Code)
					if codePreview != "" {
						evalNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Code: %s", codePreview)))
					}

					appNode.Child(evalNode)

				case StarlarkEvaluator:
					evalNode := tree.New().Root(fancy.ComponentStyle.Render("Starlark Evaluator"))
					if eval.Timeout != nil {
						evalNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Timeout: %v", eval.Timeout)))
					}

					codePreview := truncateCode(eval.Code)
					if codePreview != "" {
						evalNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Code: %s", codePreview)))
					}

					appNode.Child(evalNode)

				case ExtismEvaluator:
					evalNode := tree.New().Root(fancy.ComponentStyle.Render("Extism Evaluator"))
					if eval.Entrypoint != "" {
						evalNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Entrypoint: %s", eval.Entrypoint)))
					}

					codePreview := truncateCode(eval.Code)
					if codePreview != "" {
						evalNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Code: %s", codePreview)))
					}

					appNode.Child(evalNode)
				}

				// Add static data if present
				if len(appConfig.StaticData.Data) > 0 {
					staticNode := tree.New().Root(fancy.ComponentStyle.Render("Static Data"))
					if appConfig.StaticData.MergeMode != "" {
						staticNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Merge Mode: %s", appConfig.StaticData.MergeMode)))
					}

					for k, v := range appConfig.StaticData.Data {
						staticNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("%s: %v", k, v)))
					}

					appNode.Child(staticNode)
				}

			case CompositeScriptApp:
				appNode.Child(fancy.InfoStyle.Render("Type: Composite Script"))

				// Add script references
				if len(appConfig.ScriptAppIDs) > 0 {
					scriptsRef := tree.New().Root(fancy.ComponentStyle.Render("Scripts"))
					for _, scriptID := range appConfig.ScriptAppIDs {
						scriptsRef.Child(appStyle.Render(scriptID))
					}
					appNode.Child(scriptsRef)
				}

				// Add static data if present
				if len(appConfig.StaticData.Data) > 0 {
					staticNode := tree.New().Root(fancy.ComponentStyle.Render("Static Data"))
					if appConfig.StaticData.MergeMode != "" {
						staticNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("Merge Mode: %s", appConfig.StaticData.MergeMode)))
					}

					for k, v := range appConfig.StaticData.Data {
						staticNode.Child(fancy.InfoStyle.Render(fmt.Sprintf("%s: %v", k, v)))
					}

					appNode.Child(staticNode)
				}
			}

			appsNode.Child(appNode)
		}

		root.Child(appsNode)
	}

	return t.String()
}

// truncateCode returns a preview of code (first line, truncated)
func truncateCode(code string) string {
	if code == "" {
		return ""
	}

	// Get first line
	firstLine := strings.Split(code, "\n")[0]

	// Truncate if too long
	return fancy.TruncateString(firstLine, 50)
}
