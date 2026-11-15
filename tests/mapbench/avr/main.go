// Hash benchmark for ATmega328p boards. It times with Timer1 and keeps
// interrupts off during each batch so the Timer0 tick does not skew results.
package main

import (
	"device/avr"
	"machine"
	"runtime/interrupt"
	"time"
)

var data [64]byte

// tcnt1 reads Timer1 (the low byte read latches the high byte).
func tcnt1() uint16 {
	lo := avr.TCNT1L.Get()
	hi := avr.TCNT1H.Get()
	return uint16(hi)<<8 | uint16(lo)
}

var sink uint32

type hasher struct {
	name string
	f    func(p []byte, seed uint32) uint32
}

var hashers = []hasher{
	{"fnv", fnv32},
	{"fnvfin", fnvfin32},
	{"leveldb", leveldb32},
	{"tsip32", tsip32},
	{"sip13", sip13},
}

func nop(p []byte, seed uint32) uint32 { return uint32(len(p)) + seed }

// batch runs f k times on an n byte key and returns Timer1 ticks of 0.5 us.
// The key and seed change each call so the hash is not hoisted.
//
//go:noinline
func batch(f func([]byte, uint32) uint32, n int, k uint16) uint16 {
	p := data[:n]
	var s uint32
	st := interrupt.Disable()
	t0 := tcnt1()
	for i := uint16(0); i < k; i++ {
		p[0] = byte(i)
		s += f(p, uint32(i))
	}
	t1 := tcnt1()
	interrupt.Restore(st)
	sink += s
	return t1 - t0
}

// cycles returns the median of 5 runs in CPU cycles per call.
func cycles(f func([]byte, uint32) uint32, n int) uint32 {
	// Pick k so a batch takes about 15 ms (30000 ticks) but at least one call.
	one := uint32(batch(f, n, 1)) + 1
	k := uint32(30000) / one
	if k < 1 {
		k = 1
	}
	if k > 1000 {
		k = 1000
	}
	var r [5]uint32
	for i := range r {
		t := uint32(batch(f, n, uint16(k)))
		r[i] = t * 8 / k // ticks are 8 CPU cycles
	}
	for i := 1; i < len(r); i++ {
		for j := i; j > 0 && r[j] < r[j-1]; j-- {
			r[j], r[j-1] = r[j-1], r[j]
		}
	}
	return r[2]
}

func main() {
	for i := range data {
		data[i] = byte(i*37 + 11)
	}
	// Timer1: normal mode, prescaler 8, free running.
	avr.TCCR1A.Set(0)
	avr.TCCR1B.Set(avr.TCCR1B_CS11)

	time.Sleep(3 * time.Second) // let a serial monitor attach
	for {
		println("== AVR hash benchmark: net CPU cycles per call (16 MHz clock assumed)")
		println("keylen nop fnv fnvfin leveldb tsip32 sip13")
		for _, n := range []int{4, 8, 14, 39} {
			base := cycles(nop, n)
			print(n, " ", base)
			for _, h := range hashers {
				c := cycles(h.f, n)
				if c >= base {
					c -= base
				} else {
					c = 0
				}
				print(" ", c)
			}
			println()
		}
		println("== DONE")
		if once {
			return
		}
		time.Sleep(5 * time.Second)
	}
}

var _ = machine.Serial
