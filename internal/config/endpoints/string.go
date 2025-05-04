package endpoints

import (
	"fmt"
	"strings"

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
	// Create an endpoint tree using fancy package
	tree := fancy.EndpointTree(e.ID)

	// Add listeners
	if len(e.ListenerIDs) > 0 {
		listenersStr := strings.Join(e.ListenerIDs, ", ")
		tree.AddChild(fmt.Sprintf("Listeners: %s", listenersStr))
	}

	// Add routes
	if len(e.Routes) > 0 {
		routesNode := fancy.NewComponentTree(fmt.Sprintf("Routes (%d)", len(e.Routes)))
		for i, route := range e.Routes {
			routeSubNode := fancy.NewComponentTree(fmt.Sprintf("Route %d", i+1))
			if route.Condition != nil {
				routeSubNode.AddChild(fmt.Sprintf("App: %s", route.AppID))
				routeSubNode.AddChild(fmt.Sprintf("Condition: %s = %s", 
					route.Condition.Type(),
					route.Condition.Value()))
			} else {
				routeSubNode.AddChild(fmt.Sprintf("App: %s", route.AppID))
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
