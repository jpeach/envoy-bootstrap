package hacks

import "github.com/google/uuid"

// NewVersion returns a unique version string.
func NewVersion() string {
	return uuid.New().String()
}
