package core

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ashwaniYDV/go-redis/config"
)

// resetStore clears the in-memory store between tests so eval functions
// have a deterministic starting point.
func resetStore() {
	store = make(map[string]*Obj)
}

func TestEvalPING(t *testing.T) {
	if got := string(evalPING(nil)); got != "+PONG\r\n" {
		t.Fatalf("evalPING() = %q, want +PONG", got)
	}
	if got := string(evalPING([]string{"hello"})); got != "$5\r\nhello\r\n" {
		t.Fatalf("evalPING(hello) = %q", got)
	}
	got := string(evalPING([]string{"a", "b"}))
	if !strings.HasPrefix(got, "-") || !strings.Contains(got, "wrong number of arguments") {
		t.Fatalf("evalPING with too many args = %q", got)
	}
}

func TestEvalSET(t *testing.T) {
	resetStore()

	// too few args
	got := string(evalSET([]string{"key"}))
	if !strings.HasPrefix(got, "-") {
		t.Fatalf("evalSET too few args = %q", got)
	}

	// basic set
	if got := string(evalSET([]string{"k1", "v1"})); got != "+OK\r\n" {
		t.Fatalf("evalSET basic = %q", got)
	}
	if Get("k1").Value.(string) != "v1" {
		t.Fatalf("expected v1 in store")
	}

	// set with EX
	if got := string(evalSET([]string{"k2", "v2", "EX", "60"})); got != "+OK\r\n" {
		t.Fatalf("evalSET EX = %q", got)
	}
	obj := Get("k2")
	if obj.ExpiresAt == -1 {
		t.Fatalf("expected expiry on k2")
	}

	// set with ex (lowercase)
	if got := string(evalSET([]string{"k3", "v3", "ex", "60"})); got != "+OK\r\n" {
		t.Fatalf("evalSET ex = %q", got)
	}

	// EX missing value
	got = string(evalSET([]string{"k", "v", "EX"}))
	if !strings.Contains(got, "syntax error") {
		t.Fatalf("evalSET EX missing val = %q", got)
	}

	// EX with invalid number
	got = string(evalSET([]string{"k", "v", "EX", "notanumber"}))
	if !strings.Contains(got, "value is not an integer") {
		t.Fatalf("evalSET EX bad num = %q", got)
	}

	// unknown option
	got = string(evalSET([]string{"k", "v", "XX"}))
	if !strings.Contains(got, "syntax error") {
		t.Fatalf("evalSET unknown opt = %q", got)
	}
}

func TestEvalGET(t *testing.T) {
	resetStore()

	// wrong number of args
	got := string(evalGET([]string{}))
	if !strings.HasPrefix(got, "-") {
		t.Fatalf("evalGET no args = %q", got)
	}

	// missing key
	if got := string(evalGET([]string{"missing"})); got != string(RESP_NIL) {
		t.Fatalf("evalGET missing = %q", got)
	}

	// existing key
	Put("k", NewObj("v", -1, OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR))
	if got := string(evalGET([]string{"k"})); got != "$1\r\nv\r\n" {
		t.Fatalf("evalGET existing = %q", got)
	}

	// expired key (set ExpiresAt in the past)
	Put("expired", &Obj{Value: "v", ExpiresAt: time.Now().UnixMilli() - 1000})
	// Get() proactively deletes; bypass it by using the store directly.
	store["expired"] = &Obj{Value: "v", ExpiresAt: time.Now().UnixMilli() - 1000}
	if got := string(evalGET([]string{"expired"})); got != string(RESP_NIL) {
		t.Fatalf("evalGET expired = %q", got)
	}
}

func TestEvalTTL(t *testing.T) {
	resetStore()

	// wrong number of args
	got := string(evalTTL([]string{}))
	if !strings.HasPrefix(got, "-") {
		t.Fatalf("evalTTL no args = %q", got)
	}

	// missing key -> -2
	if got := string(evalTTL([]string{"missing"})); got != string(RESP_MINUS_2) {
		t.Fatalf("evalTTL missing = %q", got)
	}

	// no expiry -> -1
	Put("forever", NewObj("v", -1, OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR))
	if got := string(evalTTL([]string{"forever"})); got != string(RESP_MINUS_1) {
		t.Fatalf("evalTTL forever = %q", got)
	}

	// future expiry -> positive seconds
	Put("soon", NewObj("v", 60_000, OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR))
	got = string(evalTTL([]string{"soon"}))
	if !strings.HasPrefix(got, ":") {
		t.Fatalf("evalTTL soon = %q", got)
	}
	// parse the integer back
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(got, ":"), "\r\n"))
	if err != nil || n < 0 || n > 60 {
		t.Fatalf("evalTTL soon ttl = %d (err=%v)", n, err)
	}

	// already expired -> -2 (bypass Get's proactive cleanup)
	store["past"] = &Obj{Value: "v", ExpiresAt: time.Now().UnixMilli() - 1000}
	if got := string(evalTTL([]string{"past"})); got != string(RESP_MINUS_2) {
		t.Fatalf("evalTTL past = %q", got)
	}
}

func TestEvalDEL(t *testing.T) {
	resetStore()

	Put("a", NewObj("1", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT))
	Put("b", NewObj("2", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT))

	got := string(evalDEL([]string{"a", "b", "missing"}))
	if got != ":2\r\n" {
		t.Fatalf("evalDEL = %q, want :2", got)
	}
	if Get("a") != nil || Get("b") != nil {
		t.Fatalf("keys not deleted")
	}
}

func TestEvalEXPIRE(t *testing.T) {
	resetStore()

	// wrong args
	got := string(evalEXPIRE([]string{"k"}))
	if !strings.HasPrefix(got, "-") {
		t.Fatalf("evalEXPIRE wrong args = %q", got)
	}

	// non-integer duration
	got = string(evalEXPIRE([]string{"k", "abc"}))
	if !strings.Contains(got, "value is not an integer") {
		t.Fatalf("evalEXPIRE bad num = %q", got)
	}

	// missing key -> 0
	if got := string(evalEXPIRE([]string{"missing", "60"})); got != string(RESP_ZERO) {
		t.Fatalf("evalEXPIRE missing = %q", got)
	}

	// existing key -> 1
	Put("k", NewObj("v", -1, OBJ_TYPE_STRING, OBJ_ENCODING_EMBSTR))
	if got := string(evalEXPIRE([]string{"k", "60"})); got != string(RESP_ONE) {
		t.Fatalf("evalEXPIRE set = %q", got)
	}
	if Get("k").ExpiresAt == -1 {
		t.Fatalf("expected expiry on k")
	}
}

func TestEvalINCR(t *testing.T) {
	resetStore()

	// wrong number of args
	got := string(evalINCR([]string{}))
	if !strings.HasPrefix(got, "-") {
		t.Fatalf("evalINCR no args = %q", got)
	}

	// missing key auto-creates at 0, then increments to 1
	if got := string(evalINCR([]string{"counter"})); got != ":1\r\n" {
		t.Fatalf("evalINCR new key = %q, want :1", got)
	}
	// subsequent increments
	if got := string(evalINCR([]string{"counter"})); got != ":2\r\n" {
		t.Fatalf("evalINCR second = %q, want :2", got)
	}

	// INCR on an existing int-encoded value set via SET
	resetStore()
	evalSET([]string{"n", "41"})
	if got := string(evalINCR([]string{"n"})); got != ":42\r\n" {
		t.Fatalf("evalINCR on SET int = %q, want :42", got)
	}

	// INCR on a non-int (EMBSTR) value should fail the encoding assert
	resetStore()
	evalSET([]string{"name", "ashwani"})
	got = string(evalINCR([]string{"name"}))
	if !strings.HasPrefix(got, "-") {
		t.Fatalf("evalINCR on string = %q, want error", got)
	}

	// INCR on a wrong-type object should fail the type assert
	resetStore()
	Put("wrong", NewObj("0", -1, 0x10, OBJ_ENCODING_INT))
	got = string(evalINCR([]string{"wrong"}))
	if !strings.HasPrefix(got, "-") {
		t.Fatalf("evalINCR on wrong type = %q, want error", got)
	}
}

func TestEvalAndRespond(t *testing.T) {
	resetStore()

	cmds := RedisCmds{
		{Cmd: "PING", Args: nil},
		{Cmd: "SET", Args: []string{"k", "v"}},
		{Cmd: "GET", Args: []string{"k"}},
		{Cmd: "TTL", Args: []string{"k"}},
		{Cmd: "EXPIRE", Args: []string{"k", "60"}},
		{Cmd: "INCR", Args: []string{"hits"}},
		{Cmd: "BGREWRITEAOF", Args: nil},
		{Cmd: "DEL", Args: []string{"k"}},
		{Cmd: "UNKNOWN", Args: nil}, // hits default branch (treated as PING)
	}

	// Point AOFFile at a temp path so BGREWRITEAOF writes safely.
	prev := config.AOFFile
	config.AOFFile = filepath.Join(t.TempDir(), "test.aof")
	defer func() { config.AOFFile = prev }()

	var buf bytes.Buffer
	EvalAndRespond(cmds, &buf)

	out := buf.String()
	// All responses should be present back-to-back in one write.
	expectedFragments := []string{
		"+PONG\r\n",          // PING
		"+OK\r\n",            // SET / BGREWRITEAOF
		"$1\r\nv\r\n",        // GET
		string(RESP_MINUS_1), // TTL (no expiry yet)
		string(RESP_ONE),     // EXPIRE
		":1\r\n",             // DEL count
		"+PONG\r\n",          // UNKNOWN -> default PING
	}
	for _, frag := range expectedFragments {
		if !strings.Contains(out, frag) {
			t.Fatalf("expected %q in pipelined response, got %q", frag, out)
		}
	}
	// SET + BGREWRITEAOF both reply with +OK → at least 2 occurrences.
	if strings.Count(out, "+OK\r\n") < 2 {
		t.Fatalf("expected at least 2 +OK replies, got %q", out)
	}
}
