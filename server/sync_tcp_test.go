package server

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ashwaniYDV/go-redis/core"
)

func TestToArrayString(t *testing.T) {
	got, err := toArrayString([]interface{}{"GET", "foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "GET" || got[1] != "foo" {
		t.Fatalf("toArrayString = %v", got)
	}
}

func TestReadCommandsPipelined(t *testing.T) {
	// Two pipelined commands written to the same buffer:
	//   PING
	//   SET k v
	input := []byte("*1\r\n$4\r\nping\r\n*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n")
	buf := bytes.NewBuffer(input)

	cmds, err := readCommands(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 cmds, got %d", len(cmds))
	}
	// Cmd should be uppercased even if the input was lowercase.
	if cmds[0].Cmd != "PING" {
		t.Fatalf("cmd[0] = %q", cmds[0].Cmd)
	}
	if cmds[1].Cmd != "SET" || len(cmds[1].Args) != 2 || cmds[1].Args[0] != "k" || cmds[1].Args[1] != "v" {
		t.Fatalf("cmd[1] = %+v", cmds[1])
	}
}

// errReader is an io.ReadWriter whose Read always fails. Used to cover the
// read-error path of readCommands.
type errReader struct{}

func (errReader) Read(p []byte) (int, error)  { return 0, bytes.ErrTooLarge }
func (errReader) Write(p []byte) (int, error) { return len(p), nil }

func TestReadCommandsReadError(t *testing.T) {
	if _, err := readCommands(errReader{}); err == nil {
		t.Fatal("expected error from readCommands")
	}
}

// emptyReader returns 0 bytes with no error, causing Decode to fail on
// empty input.
type emptyReader struct{}

func (emptyReader) Read(p []byte) (int, error)  { return 0, nil }
func (emptyReader) Write(p []byte) (int, error) { return len(p), nil }

func TestReadCommandsDecodeError(t *testing.T) {
	if _, err := readCommands(emptyReader{}); err == nil {
		t.Fatal("expected decode error from readCommands")
	}
}

func TestRespond(t *testing.T) {
	var buf bytes.Buffer
	cmds := core.RedisCmds{
		{Cmd: "PING", Args: nil},
	}
	respond(cmds, &buf)

	out := buf.String()
	if !strings.Contains(out, "+PONG\r\n") {
		t.Fatalf("respond did not write PONG, got %q", out)
	}
}
