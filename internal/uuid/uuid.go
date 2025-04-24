package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrFailedToGenerateUUID Error = "failed to generate UUID"
	ErrInvalidUUIDFormat    Error = "invalid UUID format"
	ErrInvalidUUIDLength    Error = "invalid UUID length"
	ErrInvalidUUIDHex       Error = "invalid UUID hex character"
	ErrFailedToDecodeUUID   Error = "failed to decode UUID"
)

// UUID represents a universally unique identifier (UUID) as defined by RFC 4122.
// It is a 128-bit (16-byte) array, typically used for globally unique identification.
// The UUID is often represented as a 36-character string with hexadecimal digits and hyphens.
// UUIDs are used to uniquely identify objects or entities in distributed systems or databases.
type UUID [16]byte

// NewUUID generates a new RFC4122 version 4 UUID using a cryptographic random source and returns it along with any error encountered during generation.
func NewUUID() (UUID, error) {
	var g UUID
	_, err := rand.Read(g[:])
	if err != nil {
		return g, fmt.Errorf("%w: %v", ErrFailedToGenerateUUID, err)
	}
	// RFC4122 Version 4 UUID https://datatracker.ietf.org/doc/html/rfc4122i
	// https://guid.one/guid/make
	g[6] = (g[6] & 0x0f) | 0x40
	g[8] = (g[8] & 0x3f) | 0x80
	return g, nil
}

// String converts the UUID to its standard string representation, formatted as 8-4-4-4-12 hexadecimal characters separated by dashes.
func (g UUID) String() string {
	tl := hex.EncodeToString(g[:4])
	tm := hex.EncodeToString(g[4:6])
	th := hex.EncodeToString(g[6:8])
	cs := hex.EncodeToString(g[8:10])
	node := hex.EncodeToString(g[10:])
	return fmt.Sprint(tl, "-", tm, "-", th, "-", cs, "-", node)
}

// ParseUUID parses a UUID string in the format "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
// and returns the corresponding UUID object.
func ParseUUID(s string) (UUID, error) {
	var uuid UUID

	// Remove hyphens and validate length
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return uuid, fmt.Errorf("%w: %w", ErrInvalidUUIDFormat, ErrInvalidUUIDLength)
	}

	// Check if string contains only hex characters
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return uuid, fmt.Errorf("%w: %w", ErrInvalidUUIDFormat, ErrInvalidUUIDHex)
		}
	}

	// Decode hex string to bytes
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return uuid, fmt.Errorf("%w: %w", ErrFailedToDecodeUUID, err)
	}

	// Copy bytes to UUID
	copy(uuid[:], bytes)

	return uuid, nil
}
