package apps

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/config/styles"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a string representation of an App
func (a *App) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "App %s", a.ID)

	// Add type information
	switch cfg := a.Config.(type) {
	case ScriptApp:
		fmt.Fprintf(&b, " [Script")

		// Add evaluator type
		if cfg.Evaluator != nil {
			fmt.Fprintf(&b, " using %s", cfg.Evaluator.Type())
		}

		fmt.Fprint(&b, "]")

	case CompositeScriptApp:
		fmt.Fprintf(&b, " [CompositeScript with %d scripts]", len(cfg.ScriptAppIDs))
	default:
		fmt.Fprintf(&b, " [Unknown type]")
	}

	return b.String()
}

// ToTree returns a tree visualization of this App
func (a *App) ToTree() *fancy.ComponentTree {
	// Create a component tree for this app with consistent styling
	tree := fancy.NewComponentTree(styles.AppID(a.ID))

	// Add app-specific details based on its type
	switch appConfig := a.Config.(type) {
	case ScriptApp:
		tree.AddChild("Type: Script")

		// Add evaluator type and info
		switch eval := appConfig.Evaluator.(type) {
		case RisorEvaluator:
			evalNode := fancy.NewComponentTree("Evaluator: Risor")
			codePreview := fancy.TruncateString(eval.Code, 40)
			evalNode.AddChild(fmt.Sprintf("Code: %s", codePreview))
			if eval.Timeout != nil {
				evalNode.AddChild(fmt.Sprintf("Timeout: %v", eval.Timeout.AsDuration()))
			}
			tree.AddChild(evalNode.Tree())

		case StarlarkEvaluator:
			evalNode := fancy.NewComponentTree("Evaluator: Starlark")
			codePreview := fancy.TruncateString(eval.Code, 40)
			evalNode.AddChild(fmt.Sprintf("Code: %s", codePreview))
			if eval.Timeout != nil {
				evalNode.AddChild(fmt.Sprintf("Timeout: %v", eval.Timeout.AsDuration()))
			}
			tree.AddChild(evalNode.Tree())

		case ExtismEvaluator:
			evalNode := fancy.NewComponentTree("Evaluator: Extism")
			evalNode.AddChild(fmt.Sprintf("Entrypoint: %s", eval.Entrypoint))
			codePreview := fmt.Sprintf("<%d bytes>", len(eval.Code))
			evalNode.AddChild(fmt.Sprintf("Code: %s", codePreview))
			tree.AddChild(evalNode.Tree())
		}

		// Add static data if present
		if len(appConfig.StaticData.Data) > 0 {
			dataNode := fancy.NewComponentTree("StaticData")
			dataNode.AddChild(fmt.Sprintf("MergeMode: %s", appConfig.StaticData.MergeMode))

			// Create a data entries section
			if len(appConfig.StaticData.Data) > 0 {
				dataEntriesNode := fancy.NewComponentTree(styles.FormatSection("Entries", len(appConfig.StaticData.Data)))
				for k, v := range appConfig.StaticData.Data {
					dataEntriesNode.AddChild(fmt.Sprintf("%s: %v", k, v))
				}
				dataNode.AddChild(dataEntriesNode.Tree())
			}

			tree.AddChild(dataNode.Tree())
		}

	case CompositeScriptApp:
		tree.AddChild("Type: CompositeScript")

		// Add script apps with consistent styling
		if len(appConfig.ScriptAppIDs) > 0 {
			scriptsNode := fancy.NewComponentTree(styles.FormatSection("ScriptApps", len(appConfig.ScriptAppIDs)))
			for _, scriptID := range appConfig.ScriptAppIDs {
				// Style referenced app IDs consistently
				scriptsNode.AddChild(styles.AppID(scriptID))
			}
			tree.AddChild(scriptsNode.Tree())
		}

		// Add static data if present
		if len(appConfig.StaticData.Data) > 0 {
			dataNode := fancy.NewComponentTree("StaticData")
			dataNode.AddChild(fmt.Sprintf("MergeMode: %s", appConfig.StaticData.MergeMode))

			// Create a data entries section
			if len(appConfig.StaticData.Data) > 0 {
				dataEntriesNode := fancy.NewComponentTree(styles.FormatSection("Entries", len(appConfig.StaticData.Data)))
				for k, v := range appConfig.StaticData.Data {
					dataEntriesNode.AddChild(fmt.Sprintf("%s: %v", k, v))
				}
				dataNode.AddChild(dataEntriesNode.Tree())
			}

			tree.AddChild(dataNode.Tree())
		}
	}

	return tree
}

// ToTree returns a tree visualization of a collection of Apps
func (a Apps) ToTree() *fancy.ComponentTree {
	// Use consistent section header styling
	tree := fancy.NewComponentTree(styles.FormatSection("Apps", len(a)))

	for _, app := range a {
		appTree := app.ToTree()
		tree.AddChild(appTree.Tree())
	}

	return tree
}
