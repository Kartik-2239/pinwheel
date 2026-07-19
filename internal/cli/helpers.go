package cli

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func wrap(value int, count int) int {
	if count == 0 {
		return 0
	}
	if value < 0 {
		return count - 1
	}
	if value >= count {
		return 0
	}
	return value
}
