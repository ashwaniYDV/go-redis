package core

import (
	"strings"
	"testing"
)

func TestDeduceTypeEncoding(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		wantType uint8
		wantEnc  uint8
	}{
		{"integer", "12345", OBJ_TYPE_STRING, OBJ_ENCODING_INT},
		{"negative integer", "-42", OBJ_TYPE_STRING, OBJ_ENCODING_INT},
		{"short string", "hello", OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR},
		{"44 char boundary", strings.Repeat("a", 44), OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR},
		{"long string (raw)", strings.Repeat("a", 45), OBJ_TYPE_STRING, OBJ_ENCODING_RAW},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotEnc := deduceTypeEncoding(tc.value)
			if gotType != tc.wantType || gotEnc != tc.wantEnc {
				t.Fatalf("deduceTypeEncoding(%q) = (%#x, %#x), want (%#x, %#x)",
					tc.value, gotType, gotEnc, tc.wantType, tc.wantEnc)
			}
		})
	}
}
