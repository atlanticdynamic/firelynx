package apps

import (
	"fmt"
	"strings"

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
func (a *App) ToTree() any {
	// Create a tree node for this app
	appNode := fancy.Tree()
	appNode.Root(fancy.AppText(a.ID))

	// Add app-specific details based on its type
	switch appConfig := a.Config.(type) {
	case ScriptApp:
		appNode.Child("Type: Script")

		// Add evaluator type and info
		switch eval := appConfig.Evaluator.(type) {
		case RisorEvaluator:
			evalNode := appNode.Child("Evaluator: Risor")
			codePreview := fancy.TruncateString(eval.Code, 40)
			evalNode.Child(fmt.Sprintf("Code: %s", codePreview))
			if eval.Timeout != nil {
				evalNode.Child(
					fmt.Sprintf("Timeout: %v", eval.Timeout.AsDuration()),
				)
			}
		case StarlarkEvaluator:
			evalNode := appNode.Child("Evaluator: Starlark")
			codePreview := fancy.TruncateString(eval.Code, 40)
			evalNode.Child(fmt.Sprintf("Code: %s", codePreview))
			if eval.Timeout != nil {
				evalNode.Child(
					fmt.Sprintf("Timeout: %v", eval.Timeout.AsDuration()),
				)
			}
		case ExtismEvaluator:
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
	case CompositeScriptApp:
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

	return appNode
}

// ToTree returns a tree visualization of a collection of Apps
func (a Apps) ToTree() any {
	appsTree := fancy.Tree()
	appsTree.Root(fancy.HeaderStyle.Render("Apps"))

	for _, app := range a {
		appsTree.Child(app.ToTree())
	}

	return appsTree
}
