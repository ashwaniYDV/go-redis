package core

import "errors"

// getType extracts the object type (upper 4 bits) from a packed
// TypeEncoding byte.
func getType(te uint8) uint8 {
	return te & 0b11110000
}

// getEncoding extracts the internal encoding (lower 4 bits) from a
// packed TypeEncoding byte.
func getEncoding(te uint8) uint8 {
	return te & 0b00001111
}

// assertType returns an error if the object's type does not match t.
// Used by commands that only make sense on a specific type — e.g.
// INCR is only valid on strings.
func assertType(te uint8, t uint8) error {
	if getType(te) != t {
		return errors.New("the operation is not permitted on this type")
	}
	return nil
}

// assertEncoding returns an error if the object's encoding does not
// match e. Used by commands that need a specific internal layout —
// e.g. INCR requires the value to already be int-encoded.
func assertEncoding(te uint8, e uint8) error {
	if getEncoding(te) != e {
		return errors.New("the operation is not permitted on this encoding")
	}
	return nil
}
