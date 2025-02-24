package race

// splitString splits a string after `n` unicode chartacters
func splitString(s string, n int) (start, end string) {
	i := 0
	for j := range s {
		if i == n {
			return s[:j], s[j:]
		}
		i++
	}
	return s, ""
}
