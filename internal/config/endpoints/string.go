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
		fmt.Fprintf(&b, " [Listeners: %s]", strings.Join(e.ListenerIDs, ", "))
	}

	fmt.Fprintf(&b, " (%d routes)", len(e.Routes))
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

// String returns a string representation of a Route
func (r *Route) String() string {
	var b strings.Builder

	if r.Condition != nil {
		fmt.Fprintf(&b, "Route %s:%s -> %s",
			r.Condition.Type(),
			r.Condition.Value(),
			r.AppID)
	} else {
		fmt.Fprintf(&b, "Route <no-condition> -> %s", r.AppID)
	}

	if len(r.StaticData) > 0 {
		fmt.Fprintf(&b, " (with StaticData)")
	}

	return b.String()
}

// toTree returns a styled tree node for this Route
func (r *Route) toTree() *fancy.ComponentTree {
	// Format condition info
	var conditionInfo string
	if r.Condition != nil {
		conditionInfo = fmt.Sprintf(
			"%s:%s",
			r.Condition.Type(),
			r.Condition.Value(),
		)
	} else {
		conditionInfo = "none"
	}

	text := fancy.RouteText(fmt.Sprintf(
		"Route: %s -> %s",
		conditionInfo,
		r.AppID,
	))

	return fancy.RouteTree(text)
}

// String returns a string representation of an HTTPRoute
func (r HTTPRoute) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "HTTPRoute: %s -> %s", r.Path, r.AppID)

	if len(r.StaticData) > 0 {
		fmt.Fprintf(&b, " (with StaticData)")
	}

	return b.String()
}
