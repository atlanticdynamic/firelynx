package endpoints

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/config/styles"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a string representation of an Endpoint
func (e *Endpoint) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Endpoint %s", e.ID)

	if len(e.ListenerIDs) > 0 {
		fmt.Fprintf(&b, " [Listeners: %s]", strings.Join(e.ListenerIDs, ","))
	}

	fmt.Fprintf(&b, "\nRoutes: %d", len(e.Routes))

	for i, route := range e.Routes {
		fmt.Fprintf(&b, "\n  %d. %s", i+1, route.String())
	}

	return b.String()
}

// ToTree returns a tree visualization of this Endpoint
func (e *Endpoint) ToTree() *fancy.ComponentTree {
	// Create an endpoint tree using styled endpoint ID
	tree := fancy.NewComponentTree(styles.EndpointID(e.ID))

	// Add listeners with consistent styling
	if len(e.ListenerIDs) > 0 {
		tree.AddChild(styles.ListenerRef(e.ListenerIDs))
	}

	// Add routes
	if len(e.Routes) > 0 {
		// Use a styled section header for Routes
		routesNode := fancy.NewComponentTree(styles.FormatSection("Routes", len(e.Routes)))
		for i, route := range e.Routes {
			routeSubNode := fancy.NewComponentTree(
				fancy.RouteStyle.Render(fmt.Sprintf("Route %d", i+1)),
			)
			if route.Condition != nil {
				// Style the app reference consistently
				routeSubNode.AddChild(styles.AppRef(route.AppID))
				routeSubNode.AddChild(fmt.Sprintf("Condition: %s = %s",
					route.Condition.Type(),
					route.Condition.Value()))
			} else {
				routeSubNode.AddChild(styles.AppRef(route.AppID))
				routeSubNode.AddChild("Condition: none")
			}
			routesNode.AddChild(routeSubNode.Tree())
		}
		tree.AddChild(routesNode.Tree())
	}

	return tree
}

// String returns a string representation of the Endpoints collection
func (endpoints Endpoints) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Endpoints: %d", len(endpoints))

	for i, endpoint := range endpoints {
		fmt.Fprintf(&b, "\n%d. %s", i+1, endpoint.String())
		for j, route := range endpoint.Routes {
			fmt.Fprintf(&b, "\n   %d.%d %s", i+1, j+1, route.String())
		}
	}

	return b.String()
}
