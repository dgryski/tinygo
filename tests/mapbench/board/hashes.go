package main

import (
	"math/bits"
)

// Copies of the runtime hashes plus SipHash, so one image can time them all.
// Each takes the 32 bit runtime seed and returns 32 bits.

func fnv32(p []byte, seed uint32) uint32 {
	r := uint32(2166136261) ^ seed
	for _, c := range p {
		r ^= uint32(c)
		r *= 16777619
	}
	return r
}

// fnvfin32 is fnv32 with an extra mixing step.
func fnvfin32(p []byte, seed uint32) uint32 {
	h := fnv32(p, seed)
	h ^= h >> 15
	h *= 0x2c1b3c6d
	h ^= h >> 12
	return h
}

func leveldb32(b []byte, seed uint32) uint32 {
	const (
		lseed = 0xbc9f1d34
		m     = 0xc6a4a793
	)
	h := uint32(lseed^seed) ^ uint32(uint(len(b))*uint(m))
	for ; len(b) >= 4; b = b[4:] {
		h += uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
		h *= m
		h ^= h >> 16
	}
	switch len(b) {
	case 3:
		h += uint32(b[2]) << 16
		fallthrough
	case 2:
		h += uint32(b[1]) << 8
		fallthrough
	case 1:
		h += uint32(b[0])
		h *= m
		h ^= h >> 24
	}
	return h
}

type sip32 struct{ v0, v1 uint32 }

func (s *sip32) round() {
	s.v0 += s.v1
	s.v1 = bits.RotateLeft32(s.v1, 5) ^ s.v0
	s.v0 = bits.RotateLeft32(s.v0, 8) + s.v1
	s.v1 = bits.RotateLeft32(s.v1, 13) ^ s.v0
	s.v0 = bits.RotateLeft32(s.v0, 7)
}

func tsip32(p []byte, seed uint32) uint32 {
	s := sip32{v0: seed ^ 0x74656462, v1: 0x6c796765}
	b := uint32(len(p)) << 24
	for len(p) >= 4 {
		m := uint32(p[0]) | uint32(p[1])<<8 | uint32(p[2])<<16 | uint32(p[3])<<24
		s.v1 ^= m
		s.round()
		s.v0 ^= m
		p = p[4:]
	}
	switch len(p) {
	case 3:
		b |= uint32(p[2]) << 16
		fallthrough
	case 2:
		b |= uint32(p[1]) << 8
		fallthrough
	case 1:
		b |= uint32(p[0])
	}
	s.v1 ^= b
	s.round()
	s.v0 ^= b
	s.v1 ^= 0xff
	s.round()
	s.v1 = bits.RotateLeft32(s.v1, 16)
	s.round()
	s.v1 = bits.RotateLeft32(s.v1, 16)
	return s.v0 ^ s.v1
}

type sip64 struct{ v0, v1 uint64 }

func (s *sip64) round() {
	s.v0 += s.v1
	s.v1 = bits.RotateLeft64(s.v1, 13) ^ s.v0
	s.v0 = bits.RotateLeft64(s.v0, 35) + s.v1
	s.v1 = bits.RotateLeft64(s.v1, 17) ^ s.v0
	s.v0 = bits.RotateLeft64(s.v0, 21)
}

func tsip64(p []byte, seed uint32) uint32 {
	s := sip64{v0: uint64(seed) ^ 0x736f6d6570736575, v1: 0x646f72616e646f6d}
	b := uint64(len(p)) << 56
	for len(p) >= 8 {
		m := uint64(p[0]) | uint64(p[1])<<8 | uint64(p[2])<<16 | uint64(p[3])<<24 | uint64(p[4])<<32 | uint64(p[5])<<40 | uint64(p[6])<<48 | uint64(p[7])<<56
		s.v1 ^= m
		s.round()
		s.v0 ^= m
		p = p[8:]
	}
	for i := len(p) - 1; i >= 0; i-- {
		b |= uint64(p[i]) << (8 * uint(i))
	}
	s.v1 ^= b
	s.round()
	s.v0 ^= b
	s.v1 ^= 0xff
	s.round()
	s.v1 = bits.RotateLeft64(s.v1, 32)
	s.round()
	s.v1 = bits.RotateLeft64(s.v1, 32)
	return uint32(s.v0 ^ s.v1)
}

// SipHash-c-d with k0 as the seed and k1 as zero. Rounds are unrolled.
func sip13(p []byte, seed uint32) uint32 {
	v0 := uint64(seed) ^ 0x736f6d6570736575
	v1 := uint64(0x646f72616e646f6d)
	v2 := uint64(seed) ^ 0x6c7967656e657261
	v3 := uint64(0x7465646279746573)
	b := uint64(len(p)) << 56
	for len(p) >= 8 {
		m := uint64(p[0]) | uint64(p[1])<<8 | uint64(p[2])<<16 | uint64(p[3])<<24 | uint64(p[4])<<32 | uint64(p[5])<<40 | uint64(p[6])<<48 | uint64(p[7])<<56
		v3 ^= m
		v0 += v1
		v1 = bits.RotateLeft64(v1, 13) ^ v0
		v0 = bits.RotateLeft64(v0, 32)
		v2 += v3
		v3 = bits.RotateLeft64(v3, 16) ^ v2
		v0 += v3
		v3 = bits.RotateLeft64(v3, 21) ^ v0
		v2 += v1
		v1 = bits.RotateLeft64(v1, 17) ^ v2
		v2 = bits.RotateLeft64(v2, 32)
		v0 ^= m
		p = p[8:]
	}
	for i := len(p) - 1; i >= 0; i-- {
		b |= uint64(p[i]) << (8 * uint(i))
	}
	v3 ^= b
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 ^= b
	v2 ^= 0xff
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	return uint32(v0 ^ v1 ^ v2 ^ v3)
}

func sip24(p []byte, seed uint32) uint32 {
	v0 := uint64(seed) ^ 0x736f6d6570736575
	v1 := uint64(0x646f72616e646f6d)
	v2 := uint64(seed) ^ 0x6c7967656e657261
	v3 := uint64(0x7465646279746573)
	b := uint64(len(p)) << 56
	for len(p) >= 8 {
		m := uint64(p[0]) | uint64(p[1])<<8 | uint64(p[2])<<16 | uint64(p[3])<<24 | uint64(p[4])<<32 | uint64(p[5])<<40 | uint64(p[6])<<48 | uint64(p[7])<<56
		v3 ^= m
		v0 += v1
		v1 = bits.RotateLeft64(v1, 13) ^ v0
		v0 = bits.RotateLeft64(v0, 32)
		v2 += v3
		v3 = bits.RotateLeft64(v3, 16) ^ v2
		v0 += v3
		v3 = bits.RotateLeft64(v3, 21) ^ v0
		v2 += v1
		v1 = bits.RotateLeft64(v1, 17) ^ v2
		v2 = bits.RotateLeft64(v2, 32)
		v0 += v1
		v1 = bits.RotateLeft64(v1, 13) ^ v0
		v0 = bits.RotateLeft64(v0, 32)
		v2 += v3
		v3 = bits.RotateLeft64(v3, 16) ^ v2
		v0 += v3
		v3 = bits.RotateLeft64(v3, 21) ^ v0
		v2 += v1
		v1 = bits.RotateLeft64(v1, 17) ^ v2
		v2 = bits.RotateLeft64(v2, 32)
		v0 ^= m
		p = p[8:]
	}
	for i := len(p) - 1; i >= 0; i-- {
		b |= uint64(p[i]) << (8 * uint(i))
	}
	v3 ^= b
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 ^= b
	v2 ^= 0xff
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	v0 += v1
	v1 = bits.RotateLeft64(v1, 13) ^ v0
	v0 = bits.RotateLeft64(v0, 32)
	v2 += v3
	v3 = bits.RotateLeft64(v3, 16) ^ v2
	v0 += v3
	v3 = bits.RotateLeft64(v3, 21) ^ v0
	v2 += v1
	v1 = bits.RotateLeft64(v1, 17) ^ v2
	v2 = bits.RotateLeft64(v2, 32)
	return uint32(v0 ^ v1 ^ v2 ^ v3)
}
