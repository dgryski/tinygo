package mapbench

import "testing"

type compositeKey struct {
	S string
	N int32
}

var intSink int

const getKeys = 128

// Get benchmarks look up every key in turn.

func BenchmarkMapStringShortGet(b *testing.B) {
	keys := wordKeys(getKeys)
	m := make(map[string]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	b.ResetTimer()
	j := 0
	for i := 0; i < b.N; i++ {
		intSink += m[keys[j]]
		if j++; j == len(keys) {
			j = 0
		}
	}
}

func BenchmarkMapStringLongGet(b *testing.B) {
	words := wordKeys(getKeys)
	keys := make([]string, len(words))
	m := make(map[string]int, len(words))
	for i, w := range words {
		keys[i] = "this-is-a-longer-key-for-testing-" + w
		m[keys[i]] = i
	}
	b.ResetTimer()
	j := 0
	for i := 0; i < b.N; i++ {
		intSink += m[keys[j]]
		if j++; j == len(keys) {
			j = 0
		}
	}
}

func BenchmarkMapHeaderGet(b *testing.B) {
	keys := headerKeys(getKeys)
	m := make(map[string]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	b.ResetTimer()
	j := 0
	for i := 0; i < b.N; i++ {
		intSink += m[keys[j]]
		if j++; j == len(keys) {
			j = 0
		}
	}
}

func compositeKeys(n int) []compositeKey {
	ws := wordKeys(n)
	ns := randInts(n)
	keys := make([]compositeKey, n)
	for i := range keys {
		keys[i] = compositeKey{S: ws[i], N: int32(ns[i])}
	}
	return keys
}

func BenchmarkMapCompositeGet(b *testing.B) {
	keys := compositeKeys(getKeys)
	m := make(map[compositeKey]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	b.ResetTimer()
	j := 0
	for i := 0; i < b.N; i++ {
		intSink += m[keys[j]]
		if j++; j == len(keys) {
			j = 0
		}
	}
}

func BenchmarkMapIntGet(b *testing.B) {
	keys := randInts(getKeys)
	m := make(map[int]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	b.ResetTimer()
	j := 0
	for i := 0; i < b.N; i++ {
		intSink += m[keys[j]]
		if j++; j == len(keys) {
			j = 0
		}
	}
}

const setKeys = 1024

// Set benchmarks refill a cleared map with setKeys keys per iteration.

func BenchmarkMapCompositeSet(b *testing.B) {
	keys := compositeKeys(setKeys)
	m := make(map[compositeKey]int, len(keys))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clear(m)
		for j, k := range keys {
			m[k] = j
		}
	}
}

type bigKey [256]byte

func bigKeys(n int) []bigKey {
	rng := splitmix64(99)
	keys := make([]bigKey, n)
	for i := range keys {
		for j := 0; j < len(keys[i]); j += 8 {
			v := rng.next()
			for k := 0; k < 8; k++ {
				keys[i][j+k] = byte(v >> (8 * k))
			}
		}
	}
	return keys
}

func BenchmarkMapBigKeyGet(b *testing.B) {
	keys := bigKeys(getKeys)
	m := make(map[bigKey]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	b.ResetTimer()
	j := 0
	for i := 0; i < b.N; i++ {
		intSink += m[keys[j]]
		if j++; j == len(keys) {
			j = 0
		}
	}
}

func BenchmarkMapBigKeySet(b *testing.B) {
	keys := bigKeys(setKeys)
	m := make(map[bigKey]int, len(keys))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clear(m)
		for j := range keys {
			m[keys[j]] = j
		}
	}
}
