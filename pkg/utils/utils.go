package utils

import (
	"encoding/json"
)

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

func Concat[T any](slices ...[]T) []T {
	capacity := 0
	for _, slice := range slices {
		capacity += len(slice)
	}

	s := make([]T, 0, capacity)
	for _, slice := range slices {
		s = append(s, slice...)
	}

	return s
}

func Jsonify(data any) (map[string]any, error) {
	var out map[string]any

	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}
