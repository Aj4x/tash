package uuid

import (
	"errors"
	"testing"
)

func TestUUID_String(t *testing.T) {
	tests := []struct {
		name     string
		input    UUID
		expected string
	}{
		{
			name:     "zero UUID",
			input:    UUID{},
			expected: "00000000-0000-0000-0000-000000000000",
		},
		{
			name:     "non-zero UUID",
			input:    UUID{0x33, 0x14, 0x95, 0xaa, 0xcd, 0xef, 0x40, 0x42, 0x81, 0x23, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			expected: "331495aa-cdef-4042-8123-aabbccddeeff",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := test.input.String(); result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestNewUUID(t *testing.T) {
	t.Run("generate UUID", func(t *testing.T) {
		uuid, err := NewUUID()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if (uuid[6] & 0xf0) != 0x40 {
			t.Errorf("expected version 4, but got different version")
		}
		if (uuid[8] & 0xc0) != 0x80 {
			t.Errorf("expected variant field to indicate RFC 4122, but got different value")
		}
	})
}

func TestParseUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errs     []error
		expected UUID
	}{
		{
			name:     "valid UUID",
			input:    "331495AA-CDEF-4042-8123-AABBCCDDEEFF",
			wantErr:  false,
			errs:     []error{},
			expected: UUID{0x33, 0x14, 0x95, 0xaa, 0xcd, 0xef, 0x40, 0x42, 0x81, 0x23, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		},
		{
			name:     "valid UUID without hyphens",
			input:    "331495AACDEF40428123AABBCCDDEEFF",
			wantErr:  false,
			errs:     []error{},
			expected: UUID{0x33, 0x14, 0x95, 0xaa, 0xcd, 0xef, 0x40, 0x42, 0x81, 0x23, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		},
		{
			name:    "invalid length",
			input:   "12345678-1234-1234-1234",
			wantErr: true,
			errs:    []error{ErrInvalidUUIDFormat, ErrInvalidUUIDLength},
		},
		{
			name:    "non-hexadecimal character",
			input:   "zz1495AA-CDEF-4042-8123-AABBCCDDEEFF",
			wantErr: true,
			errs:    []error{ErrInvalidUUIDFormat, ErrInvalidUUIDHex},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			uuid, err := ParseUUID(test.input)
			if test.wantErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				}
				for _, e := range test.errs {
					if !errors.Is(err, e) {
						t.Errorf("expected error %v, got %v", e, err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if uuid != test.expected {
					t.Errorf("expected UUID %v, got %v", test.expected, uuid)
				}
			}
		})
	}
}
