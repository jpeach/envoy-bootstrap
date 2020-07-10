package hacks

import "math/rand"

const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandomStringN ...
func RandomStringN(length int) string {
	if length < 1 {
		return ""
	}

	result := make([]byte, length)

	for i := range result {
		result[i] = alpha[rand.Int()%len(alpha)] //nolint(gosec)
	}

	return string(result)
}
