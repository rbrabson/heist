package heist

// comparable defines the objects that `contains` and `remove` may function on.
type comparable interface {
	string | *Player
}

// contains determines if the element is in the slice.
func contains[V comparable](list []V, element V) bool {
	for _, a := range list {
		if a == element {
			return true
		}
	}
	return false
}
