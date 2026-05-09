package config

var Host string = "0.0.0.0"
var Port int = 7379
var KeysLimit int = 100

// EvictionRatio dictates the fraction of KeysLimit that is evicted
// whenever eviction is triggered. e.g. 0.40 evicts 40% of the keys.
var EvictionRatio float64 = 0.40

var EvictionStrategy string = "allkeys-random"

// AOFFile is the on-disk location where the Append-Only File is
// written. On BGREWRITEAOF the entire dataset is dumped here as a
// sequence of RESP-encoded commands.
var AOFFile string = "./go-redis-master.aof"
