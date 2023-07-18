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

// remove removes an element from the slice.
func remove[V comparable](list []V, element V) []V {
	ret := make([]V, 0)
	for i, a := range list {
		if a == element {
			ret = append(ret, list[:i]...)
			ret = append(ret, list[i+1:]...)
		}
	}
	return ret
}
