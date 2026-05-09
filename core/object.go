package core

// TODO: change ExpiresAt it to LRU Bits as handled by Redis
type Obj struct {
	// TypeEncoding packs two pieces of information into a single byte.
	//
	// Redis tracks both the data type (string, list, set, ...) and the
	// internal encoding used to store it (raw, int, embstr, ...). In the
	// Redis C source these are two separate 4-bit fields on robj:
	//
	//     unsigned type:4;      // upper 4 bits -> object type
	//     unsigned encoding:4;  // lower 4 bits -> internal encoding
	//
	// Go has no uint4 type, so we cannot declare two 4-bit fields.
	// Instead we use one uint8 and split it ourselves: the upper 4 bits
	// hold the type and the lower 4 bits hold the encoding. They are
	// combined with a bitwise OR (oType | oEnc) and read back via masks
	// (see getType / getEncoding in typeencoding.go).
	TypeEncoding uint8
	Value        interface{}
	ExpiresAt    int64
}

var OBJ_TYPE_STRING uint8 = 0 << 4

var OBJ_ENCODING_RAW uint8 = 0
var OBJ_ENCODING_INT uint8 = 1
var OBJ_ENCODING_EMBSTR uint8 = 8
