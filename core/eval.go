package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

// Pre-encoded RESP responses for the most common one-shot replies.
// Reusing these byte slices avoids re-allocating the same response on
// every command and keeps the hot path allocation-free.
var RESP_NIL []byte = []byte("$-1\r\n")
var RESP_OK []byte = []byte("+OK\r\n")
var RESP_ZERO []byte = []byte(":0\r\n")
var RESP_ONE []byte = []byte(":1\r\n")
var RESP_MINUS_1 []byte = []byte(":-1\r\n")
var RESP_MINUS_2 []byte = []byte(":-2\r\n")

// evalPING handles the PING command. With no args it returns "PONG";
// with one arg it echoes the arg back; more than one arg is an error.
func evalPING(args []string) []byte {
	var b []byte

	if len(args) >= 2 {
		return Encode(errors.New("ERR wrong number of arguments for 'ping' command"), false)
	}

	if len(args) == 0 {
		b = Encode("PONG", true)
	} else {
		b = Encode(args[0], false)
	}

	return b
}

// evalSET stores key=value, optionally with an EX <seconds> TTL.
// Returns +OK on success or a RESP error for bad arity / bad options.
func evalSET(args []string) []byte {
	if len(args) <= 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'set' command"), false)
	}

	var key, value string
	var exDurationMs int64 = -1

	key, value = args[0], args[1]
	oType, oEnc := deduceTypeEncoding(value)

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			i++
			if i == len(args) {
				return Encode(errors.New("(error) ERR syntax error"), false)
			}

			exDurationSec, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return Encode(errors.New("(error) ERR value is not an integer or out of range"), false)
			}
			exDurationMs = exDurationSec * 1000
		default:
			return Encode(errors.New("(error) ERR syntax error"), false)
		}
	}

	// putting the k and value in a Hash Table
	Put(key, NewObj(value, exDurationMs, oType, oEnc))
	return RESP_OK
}

// evalGET returns the value for a key, or RESP nil if the key is
// missing or has expired.
func evalGET(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}

	var key string = args[0]

	// Get the key from the hash table
	obj := Get(key)

	// if key does not exist, return RESP encoded nil
	if obj == nil {
		return RESP_NIL
	}

	// if key already expired then return nil
	if obj.ExpiresAt != -1 && obj.ExpiresAt <= time.Now().UnixMilli() {
		return RESP_NIL
	}

	// return the RESP encoded value
	return Encode(obj.Value, false)
}

// evalTTL returns the remaining TTL of a key in seconds.
// -2 if the key doesn't exist, -1 if the key has no expiration set.
func evalTTL(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'ttl' command"), false)
	}

	var key string = args[0]

	obj := Get(key)

	// if key does not exist, return RESP encoded -2 denoting key does not exist
	if obj == nil {
		return RESP_MINUS_2
	}

	// if object exist, but no expiration is set on it then send -1
	if obj.ExpiresAt == -1 {
		return RESP_MINUS_1
	}

	// compute the time remaining for the key to expire and
	// return the RESP encoded form of it
	durationMs := obj.ExpiresAt - time.Now().UnixMilli()

	// if key expired i.e. key does not exist hence return -2
	if durationMs < 0 {
		return RESP_MINUS_2
	}

	return Encode(int64(durationMs/1000), false)
}

// evalDEL removes each given key and returns the number of keys
// actually deleted (missing keys are silently skipped).
func evalDEL(args []string) []byte {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}

	return Encode(countDeleted, false)
}

// evalBGREWRITEAOF triggers a synchronous AOF rewrite — the whole
// dataset is dumped to disk as RESP-encoded SET commands so the file
// can be replayed on next bootup.
//
// TODO: Make it async by forking a new process. The real Redis
// BGREWRITEAOF returns immediately and does the work in a child.
func evalBGREWRITEAOF(args []string) []byte {
	DumpAllAOF()
	return RESP_OK
}

func evalINCR(args []string) []byte {
	if len(args) != 1 {
		return Encode(errors.New("ERR wrong number of arguments for 'incr' command"), false)
	}

	var key string = args[0]
	obj := Get(key)
	if obj == nil {
		obj = NewObj("0", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj)
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		return Encode(err, false)
	}

	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		return Encode(err, false)
	}

	i, _ := strconv.ParseInt(obj.Value.(string), 10, 64)
	i++
	obj.Value = strconv.FormatInt(i, 10)

	return Encode(i, false)
}

// evalEXPIRE attaches a TTL (in seconds) to an existing key.
// Returns 1 if the timeout was set, 0 if the key doesn't exist.
func evalEXPIRE(args []string) []byte {
	if len(args) <= 1 {
		return Encode(errors.New("(error) ERR wrong number of arguments for 'expire' command"), false)
	}

	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return Encode(errors.New("(error) ERR value is not an integer or out of range"), false)
	}

	obj := Get(key)

	// 0 if the timeout was not set. e.g. key doesn't exist, or operation skipped due to the provided arguments
	if obj == nil {
		return RESP_ZERO
	}

	obj.ExpiresAt = time.Now().UnixMilli() + exDurationSec*1000

	// 1 if the timeout was set.
	return RESP_ONE
}

// evalINFO returns server statistics in the same format real Redis
// uses, so existing tooling (redis_exporter, redis-cli) can read it.
// We only populate the "# Keyspace" section: one line per database
// reporting the number of keys it currently holds.
func evalINFO(args []string) []byte {
	var info []byte
	buf := bytes.NewBuffer(info)
	buf.WriteString("# Keyspace\r\n")
	for i := range KeyspaceStat {
		buf.WriteString(fmt.Sprintf("db%d:keys=%d,expires=0,avg_ttl=0\r\n", i, KeyspaceStat[i]["keys"]))
	}
	return Encode(buf.String(), false)
}

// evalCLIENT is a stub for the CLIENT command. Real Redis exposes many
// subcommands (LIST, SETNAME, ID, ...); tooling like redis-cli sends
// CLIENT on connect, so we just acknowledge it with +OK.
func evalCLIENT(args []string) []byte {
	return RESP_OK
}

// evalLATENCY is a stub for the LATENCY command. Real Redis reports
// latency spikes; we have nothing to track yet, so we return an empty
// array which is a valid reply for LATENCY HISTORY / LATENCY LATEST.
func evalLATENCY(args []string) []byte {
	return Encode([]string{}, false)
}

// EvalAndRespond evaluates each command in the pipeline, accumulates
// every response in a single buffer, and writes the entire batch to
// the connection in one Write call. Batching the responses is what
// makes pipelining cheap: one syscall per pipeline instead of one per
// command.
func EvalAndRespond(cmds RedisCmds, c io.ReadWriter) {
	// Single buffer collects every command's reply in order.
	var response []byte
	buf := bytes.NewBuffer(response)

	for _, cmd := range cmds {
		switch cmd.Cmd {
		case "PING":
			buf.Write(evalPING(cmd.Args))
		case "SET":
			buf.Write(evalSET(cmd.Args))
		case "GET":
			buf.Write(evalGET(cmd.Args))
		case "TTL":
			buf.Write(evalTTL(cmd.Args))
		case "DEL":
			buf.Write(evalDEL(cmd.Args))
		case "EXPIRE":
			buf.Write(evalEXPIRE(cmd.Args))
		case "BGREWRITEAOF":
			buf.Write(evalBGREWRITEAOF(cmd.Args))
		case "INCR":
			buf.Write(evalINCR(cmd.Args))
		case "INFO":
			buf.Write(evalINFO(cmd.Args))
		case "CLIENT":
			buf.Write(evalCLIENT(cmd.Args))
		case "LATENCY":
			buf.Write(evalLATENCY(cmd.Args))
		default:
			// Unknown command falls back to PING so we never crash on
			// a typo; clients still get a well-formed RESP reply.
			buf.Write(evalPING(cmd.Args))
		}
	}

	// Single Write flushes all replies for the batch at once.
	c.Write(buf.Bytes())
}
