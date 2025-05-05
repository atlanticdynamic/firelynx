package routes

import (
	"fmt"
	"sort"
	"strings"
)

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
		fmt.Fprintf(&b, " (with StaticData: ")
		keys := make([]string, 0, len(r.StaticData))
		for k := range r.StaticData {
			keys = append(keys, k)
		}
		// Sort keys for predictable output
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				fmt.Fprintf(&b, ", ")
			}
			fmt.Fprintf(&b, "%s=%v", k, r.StaticData[k])
		}
		fmt.Fprintf(&b, ")")
	}

	return b.String()
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
