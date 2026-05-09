package core

import "github.com/ashwaniYDV/go-redis/config"

// Evicts the first key it found while iterating the map
// TODO: Make it efficient by doing thorough sampling
func evictFirst() {
	for k := range store {
		Del(k)
		return
	}
}

// evictAllkeysRandom randomly removes keys to make space for new data.
// The number of keys removed is EvictionRatio * KeysLimit.
//
// Go does not guarantee a random map iteration, but because each key's
// slot is determined by its hash, iterating the map gives a fairly
// random distribution for our purposes.
func evictAllkeysRandom() {
	evictCount := int64(config.EvictionRatio * float64(config.KeysLimit))
	for k := range store {
		Del(k)
		evictCount--
		if evictCount <= 0 {
			break
		}
	}
}

// TODO: Make the eviction strategy configuration driven
// TODO: Support multiple eviction strategies
func evict() {
	switch config.EvictionStrategy {
	case "simple-first":
		evictFirst()
	case "allkeys-random":
		evictAllkeysRandom()
	}
}
