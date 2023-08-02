package utils

func Contains[T comparable](slice []T, comp T) bool {
	for _, val := range slice {
		if val == comp {
			return true
		}
	}

	return false
}

func IntPow(n, m int) int {
	if m == 0 {
		return 1
	}
	result := n
	for i := 2; i <= m; i++ {
		result *= n
	}
	return result
}
