package core

import "testing"

func TestGetType(t *testing.T) {
	// STRING type (high nibble 0) + INT encoding (low nibble 1) = 0x01
	if got := getType(OBJ_TYPE_STRING | OBJ_ENCODING_INT); got != OBJ_TYPE_STRING {
		t.Fatalf("getType = %#x, want %#x", got, OBJ_TYPE_STRING)
	}
	// A non-zero high nibble should be returned as the type.
	var fakeType uint8 = 0x10
	if got := getType(fakeType | OBJ_ENCODING_EMBSTR); got != fakeType {
		t.Fatalf("getType high-nibble = %#x, want %#x", got, fakeType)
	}
}

func TestGetEncoding(t *testing.T) {
	cases := []struct {
		te   uint8
		want uint8
	}{
		{OBJ_TYPE_STRING | OBJ_ENCODING_RAW, OBJ_ENCODING_RAW},
		{OBJ_TYPE_STRING | OBJ_ENCODING_INT, OBJ_ENCODING_INT},
		{OBJ_TYPE_STRING | OBJ_ENCODING_EMBSTR, OBJ_ENCODING_EMBSTR},
	}
	for _, tc := range cases {
		if got := getEncoding(tc.te); got != tc.want {
			t.Fatalf("getEncoding(%#x) = %#x, want %#x", tc.te, got, tc.want)
		}
	}
}

func TestAssertType(t *testing.T) {
	te := OBJ_TYPE_STRING | OBJ_ENCODING_INT
	if err := assertType(te, OBJ_TYPE_STRING); err != nil {
		t.Fatalf("assertType matching: %v", err)
	}
	var otherType uint8 = 0x10
	if err := assertType(te, otherType); err == nil {
		t.Fatal("assertType mismatching: expected error")
	}
}

func TestAssertEncoding(t *testing.T) {
	te := OBJ_TYPE_STRING | OBJ_ENCODING_INT
	if err := assertEncoding(te, OBJ_ENCODING_INT); err != nil {
		t.Fatalf("assertEncoding matching: %v", err)
	}
	if err := assertEncoding(te, OBJ_ENCODING_RAW); err == nil {
		t.Fatal("assertEncoding mismatching: expected error")
	}
}
