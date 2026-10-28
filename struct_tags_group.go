package dig

import "strings"

type group struct {
	name    string
	flatten bool
	passive bool
}

var _ func(string) (group, error) = parseGroupString

func parseGroupString(s string) (group, error) {
	components := strings.Split(s, ",")
	g := group{name: components[0]}
	for _, c := range components[1:] {
		switch c {
		case "flatten":
			g.flatten = true
		case "passive":
			g.passive = true
		default:
			return g, errInvalidGroupOption{Option: c}
		}
	}
	return g, nil
}
