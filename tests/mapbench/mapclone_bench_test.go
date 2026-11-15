package mapbench

import (
	"maps"
	"strconv"
	"testing"
)

type bigVal [200]byte

var (
	cloneIntSink map[int]int
	cloneStrSink map[string]int
	cloneBigSink map[int]bigVal
)

var cloneSizes = []int{4, 64, 1024, 16384}

// Keys are random ints, short English words and HTTP header names.

func BenchmarkMapCloneInt(b *testing.B) {
	for _, n := range cloneSizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			keys := randInts(n)
			m := make(map[int]int, n)
			for i, k := range keys {
				m[k] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cloneIntSink = maps.Clone(m)
			}
		})
	}
}

func BenchmarkMapCloneString(b *testing.B) {
	for _, n := range cloneSizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			m := make(map[string]int, n)
			for i, k := range wordKeys(n) {
				m[k] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cloneStrSink = maps.Clone(m)
			}
		})
	}
}

func BenchmarkMapCloneHeader(b *testing.B) {
	for _, n := range cloneSizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			m := make(map[string]int, n)
			for i, k := range headerKeys(n) {
				m[k] = i
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cloneStrSink = maps.Clone(m)
			}
		})
	}
}

// Large values are stored indirectly, so each entry needs its own allocation.
func BenchmarkMapCloneBigValue(b *testing.B) {
	for _, n := range cloneSizes[:3] {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			m := make(map[int]bigVal, n)
			for i, k := range randInts(n) {
				m[k] = bigVal{byte(i)}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cloneBigSink = maps.Clone(m)
			}
		})
	}
}

// Sparse map: many deletes leave mostly empty buckets to walk.
func BenchmarkMapCloneSparse(b *testing.B) {
	keys := randInts(16384)
	m := make(map[int]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	for i, k := range keys {
		if i%16 != 0 {
			delete(m, k)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cloneIntSink = maps.Clone(m)
	}
}

// Growing from empty runs hashmapCopy at every doubling.
func BenchmarkMapGrowInt(b *testing.B) {
	for _, n := range cloneSizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			keys := randInts(n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m := map[int]int{}
				for j, k := range keys {
					m[k] = j
				}
				cloneIntSink = m
			}
		})
	}
}

func BenchmarkMapGrowString(b *testing.B) {
	keys := wordKeys(1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := map[string]int{}
		for j, k := range keys {
			m[k] = j
		}
		cloneStrSink = m
	}
}

func BenchmarkMapGrowHeader(b *testing.B) {
	keys := headerKeys(1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := map[string]int{}
		for j, k := range keys {
			m[k] = j
		}
		cloneStrSink = m
	}
}

func BenchmarkMapClearEmpty(b *testing.B) {
	for _, n := range []int{64, 16384} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			m := make(map[int]int, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				clear(m)
			}
		})
	}
}

func BenchmarkMapClearFull(b *testing.B) {
	for _, n := range []int{64, 16384} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			keys := randInts(n)
			m := make(map[int]int, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j, k := range keys {
					m[k] = j
				}
				clear(m)
			}
		})
	}
}
