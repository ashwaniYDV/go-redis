package core

import (
	"time"

	"github.com/ashwaniYDV/go-redis/config"
)

var store map[string]*Obj

func init() {
	store = make(map[string]*Obj)
}

func NewObj(value interface{}, durationMs int64, oType uint8, oEnc uint8) *Obj {
	var expiresAt int64 = -1
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + durationMs
	}

	return &Obj{
		Value:        value,
		TypeEncoding: oType | oEnc,
		ExpiresAt:    expiresAt,
	}
}

func Put(k string, obj *Obj) {
	if len(store) >= config.KeysLimit {
		evict()
	}
	store[k] = obj
	// Record the actual map size rather than incrementing a counter.
	// Incrementing on every Put would overcount when an existing key
	// is overwritten; len(store) is always the absolute, correct count.
	UpdateDBStat(0, "keys", len(store))
}

func Get(k string) *Obj {
	v := store[k]
	if v != nil {
		// if the key already expired then delete it and return nil
		if v.ExpiresAt != -1 && v.ExpiresAt <= time.Now().UnixMilli() {
			delete(store, k)
			UpdateDBStat(0, "keys", len(store))
			return nil
		}
	}
	return v
}

func Del(k string) bool {
	if _, ok := store[k]; ok {
		delete(store, k)
		UpdateDBStat(0, "keys", len(store))
		return true
	}
	return false
}
