package core

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ashwaniYDV/go-redis/config"
)

// dumpKey writes a single key/value as a RESP-encoded SET command to fp,
// so the AOF file replays as plain Redis commands on bootup.
//
// TODO: Support non-kv data structures (lists, hashes, etc).
// TODO: Support sync write so callers can wait for fsync.
// TODO: Honour TTL by emitting "SET <k> <v> EX <ttl>" when ExpiresAt > 0.
func dumpKey(fp *os.File, key string, obj *Obj) {
	cmd := fmt.Sprintf("SET %s %s", key, obj.Value)
	tokens := strings.Split(cmd, " ")
	fp.Write(Encode(tokens, false))
}

// DumpAllAOF rewrites the entire current dataset to the AOF file as a
// sequence of SET commands. This is the BGREWRITEAOF compaction path:
// instead of replaying every historical write, we snapshot the live
// state so the file stays small.
//
// O_TRUNC is intentional — a rewrite replaces the file's contents.
// Without truncation, a shorter dump would leave stale tail data
// behind from the previous rewrite.
//
// TODO: Write to a temp file and atomically rename to avoid leaving
// a half-written AOF if the process dies mid-rewrite.
func DumpAllAOF() {
	fp, err := os.OpenFile(config.AOFFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("error", err)
		return
	}
	defer fp.Close()
	log.Println("rewriting AOF file at", config.AOFFile)
	for k, obj := range store {
		dumpKey(fp, k, obj)
	}
	log.Println("AOF file rewrite complete")
}
