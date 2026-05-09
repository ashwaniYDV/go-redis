package core

// RedisCmd is a single parsed Redis command (the verb + its args).
type RedisCmd struct {
	Cmd  string
	Args []string
}

// RedisCmds is a batch of commands read from one client write.
// Pipelining lets a client send several commands back-to-back, so the
// server consumes them as a slice instead of one at a time.
type RedisCmds []*RedisCmd
