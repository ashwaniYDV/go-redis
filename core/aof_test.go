package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ashwaniYDV/go-redis/config"
)

// withTempAOF points config.AOFFile at a fresh path inside t.TempDir
// for the duration of the test, restoring the original afterwards.
func withTempAOF(t *testing.T) string {
	t.Helper()
	prev := config.AOFFile
	path := filepath.Join(t.TempDir(), "test.aof")
	config.AOFFile = path
	t.Cleanup(func() { config.AOFFile = prev })
	return path
}

func TestDumpKey(t *testing.T) {
	path := withTempAOF(t)

	fp, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer fp.Close()

	dumpKey(fp, "k1", &Obj{Value: "v1", ExpiresAt: -1})

	if err := fp.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "*3\r\n$3\r\nSET\r\n$2\r\nk1\r\n$2\r\nv1\r\n"
	if string(data) != want {
		t.Fatalf("dumpKey wrote %q, want %q", data, want)
	}
}

func TestDumpAllAOF(t *testing.T) {
	resetStore()
	path := withTempAOF(t)

	Put("alpha", NewObj("1", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT))
	Put("beta", NewObj("2", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT))

	DumpAllAOF()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out := string(data)
	// map iteration order is random — assert each entry is present
	// rather than ordering.
	if !strings.Contains(out, "$5\r\nalpha\r\n") || !strings.Contains(out, "$1\r\n1\r\n") {
		t.Fatalf("alpha entry missing in AOF: %q", out)
	}
	if !strings.Contains(out, "$4\r\nbeta\r\n") || !strings.Contains(out, "$1\r\n2\r\n") {
		t.Fatalf("beta entry missing in AOF: %q", out)
	}
	// Two SET commands → two RESP arrays of arity 3.
	if strings.Count(out, "*3\r\n$3\r\nSET\r\n") != 2 {
		t.Fatalf("expected 2 SET commands in AOF: %q", out)
	}
}

func TestDumpAllAOFOpenError(t *testing.T) {
	prev := config.AOFFile
	t.Cleanup(func() { config.AOFFile = prev })
	// Pointing AOFFile at a directory that doesn't exist forces OpenFile
	// to fail and exercises the error branch of DumpAllAOF.
	config.AOFFile = filepath.Join(t.TempDir(), "no-such-dir", "x.aof")

	// Should not panic, just print the error and return.
	DumpAllAOF()
}

func TestEvalBGREWRITEAOF(t *testing.T) {
	resetStore()
	path := withTempAOF(t)

	Put("k", NewObj("v", -1, OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR))

	got := evalBGREWRITEAOF(nil)
	if string(got) != string(RESP_OK) {
		t.Fatalf("evalBGREWRITEAOF = %q, want +OK", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n"
	if string(data) != want {
		t.Fatalf("AOF after BGREWRITEAOF = %q, want %q", data, want)
	}
}
