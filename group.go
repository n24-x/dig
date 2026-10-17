package dig

import "strings"

type group struct {
	Name    string
	Flatten bool
	Passive bool
}

var _ func(string) (group, error) = parseGroupString

func parseGroupString(s string) (group, error) {
	components := strings.Split(s, ",")
	g := group{Name: components[0]}
	for _, c := range components[1:] {
		switch c {
		case "flatten":
			g.Flatten = true
		case "passive":
			g.Passive = true
		default:
			return g, errInvalidGroupOption{Option: c}
		}
	}
	return g, nil
}
