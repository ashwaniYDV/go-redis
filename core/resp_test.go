package core_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/ashwaniYDV/go-redis/core"
)

func TestSimpleStringDecode(t *testing.T) {
	cases := map[string]string{
		"+OK\r\n": "OK",
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if v != values[0] {
			t.Fail()
		}
	}
}

func TestError(t *testing.T) {
	cases := map[string]string{
		"-Error message\r\n": "Error message",
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if v != values[0] {
			t.Fail()
		}
	}
}

func TestInt64(t *testing.T) {
	cases := map[string]int64{
		":0\r\n":    0,
		":1000\r\n": 1000,
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if v != values[0] {
			t.Fail()
		}
	}
}

func TestBulkStringDecode(t *testing.T) {
	cases := map[string]string{
		"$5\r\nhello\r\n": "hello",
		"$0\r\n\r\n":      "",
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		if v != values[0] {
			t.Fail()
		}
	}
}

func TestArrayDecode(t *testing.T) {
	cases := map[string][]interface{}{
		"*0\r\n":                                                   {},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n":                     {"hello", "world"},
		"*3\r\n:1\r\n:2\r\n:3\r\n":                                 {int64(1), int64(2), int64(3)},
		"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nhello\r\n":            {int64(1), int64(2), int64(3), int64(4), "hello"},
		"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n": {[]int64{int64(1), int64(2), int64(3)}, []interface{}{"Hello", "World"}},
	}
	for k, v := range cases {
		values, _ := core.Decode([]byte(k))
		array := values[0].([]interface{})
		if len(array) != len(v) {
			t.Fail()
		}
		for i := range array {
			if fmt.Sprintf("%v", v[i]) != fmt.Sprintf("%v", array[i]) {
				t.Fail()
			}
		}
	}
}

// TestPipelinedDecode verifies that Decode returns multiple top-level
// values when several RESP messages are concatenated in the same buffer.
func TestPipelinedDecode(t *testing.T) {
	// Two pipelined commands: PING and GET foo
	data := []byte("*1\r\n$4\r\nPING\r\n*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")
	values, err := core.Decode(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 decoded values, got %d", len(values))
	}

	first := values[0].([]interface{})
	if len(first) != 1 || first[0].(string) != "PING" {
		t.Fatalf("unexpected first command: %v", first)
	}

	second := values[1].([]interface{})
	if len(second) != 2 || second[0].(string) != "GET" || second[1].(string) != "foo" {
		t.Fatalf("unexpected second command: %v", second)
	}
}

func TestDecodeEmpty(t *testing.T) {
	_, err := core.Decode([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestDecodeOneEmpty(t *testing.T) {
	_, _, err := core.DecodeOne([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestDecodeOneUnknownPrefix(t *testing.T) {
	value, delta, err := core.DecodeOne([]byte("?bogus\r\n"))
	if err == nil {
		t.Fatalf("expected error for unknown prefix, got value=%v delta=%d", value, delta)
	}
}

// TestDecodeInlineCommandDoesNotHang ensures a non-RESP "inline"
// command (plain text, no type byte) is rejected with an error instead
// of spinning forever in the decode loop. This previously hung the
// single-threaded event-loop server.
func TestDecodeInlineCommandDoesNotHang(t *testing.T) {
	_, err := core.Decode([]byte("SET a 1\r\n"))
	if err == nil {
		t.Fatal("expected error for inline (non-RESP) command")
	}
}

func TestEncodeSimpleString(t *testing.T) {
	got := core.Encode("OK", true)
	if !bytes.Equal(got, []byte("+OK\r\n")) {
		t.Fatalf("unexpected encoding: %q", got)
	}
}

func TestEncodeBulkString(t *testing.T) {
	got := core.Encode("hello", false)
	if !bytes.Equal(got, []byte("$5\r\nhello\r\n")) {
		t.Fatalf("unexpected encoding: %q", got)
	}
}

func TestEncodeStringArray(t *testing.T) {
	// "SET k v" -> RESP array of 3 bulk strings, the format AOF persists.
	got := core.Encode([]string{"SET", "k", "v"}, false)
	want := []byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Encode([]string) = %q, want %q", got, want)
	}
}

func TestEncodeEmptyStringArray(t *testing.T) {
	got := core.Encode([]string{}, false)
	if !bytes.Equal(got, []byte("*0\r\n")) {
		t.Fatalf("Encode([]string{}) = %q", got)
	}
}

func TestEncodeIntegers(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{int(1), ":1\r\n"},
		{int8(2), ":2\r\n"},
		{int16(3), ":3\r\n"},
		{int32(4), ":4\r\n"},
		{int64(5), ":5\r\n"},
	}
	for _, tc := range cases {
		got := core.Encode(tc.in, false)
		if string(got) != tc.want {
			t.Fatalf("Encode(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEncodeError(t *testing.T) {
	got := core.Encode(errors.New("boom"), false)
	if !bytes.Equal(got, []byte("-boom\r\n")) {
		t.Fatalf("unexpected encoding: %q", got)
	}
}

func TestEncodeUnsupportedType(t *testing.T) {
	got := core.Encode(3.14, false)
	if !bytes.Equal(got, core.RESP_NIL) {
		t.Fatalf("expected RESP_NIL for unsupported type, got %q", got)
	}
}
