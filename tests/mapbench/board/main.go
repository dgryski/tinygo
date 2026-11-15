// Hash benchmark for small boards. It times the map hash functions directly
// and prints net cycles per call over the default serial.
package main

import (
	"runtime"
	"strconv"
	"time"
	"unsafe"
)

type splitmix64 uint64

func (s *splitmix64) next() uint64 {
	*s += 0x9e3779b97f4a7c15
	z := uint64(*s)
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	return z ^ z>>31
}

// Header name strings alias this const, so they stay in flash.
const headerData = `Accept Accept-Charset Accept-Encoding Accept-Language Accept-Ranges
Access-Control-Allow-Credentials Access-Control-Allow-Headers
Access-Control-Allow-Methods Access-Control-Allow-Origin
Access-Control-Expose-Headers Access-Control-Max-Age
Access-Control-Request-Headers Access-Control-Request-Method Age Allow
Alt-Svc Authorization Cache-Control Clear-Site-Data Connection
Content-Disposition Content-Encoding Content-Language Content-Length
Content-Location Content-Range Content-Security-Policy
Content-Security-Policy-Report-Only Content-Type Cookie
Cross-Origin-Embedder-Policy Cross-Origin-Opener-Policy
Cross-Origin-Resource-Policy Date Device-Memory DNT Early-Data ETag
Expect Expect-CT Expires Forwarded From Host If-Match If-Modified-Since
If-None-Match If-Range If-Unmodified-Since Keep-Alive Last-Modified Link
Location Max-Forwards Origin Pragma Proxy-Authenticate
Proxy-Authorization Range Referer Referrer-Policy Retry-After
Save-Data Sec-CH-UA Sec-CH-UA-Mobile Sec-CH-UA-Platform Sec-Fetch-Dest
Sec-Fetch-Mode Sec-Fetch-Site Sec-Fetch-User Sec-WebSocket-Accept
Sec-WebSocket-Extensions Sec-WebSocket-Key Sec-WebSocket-Protocol
Sec-WebSocket-Version Server Server-Timing Set-Cookie SourceMap
Strict-Transport-Security TE Timing-Allow-Origin Trailer
Transfer-Encoding Upgrade Upgrade-Insecure-Requests User-Agent Vary Via
Want-Digest Warning WWW-Authenticate X-Content-Type-Options
X-DNS-Prefetch-Control X-Forwarded-For X-Forwarded-Host
X-Forwarded-Proto X-Frame-Options X-Powered-By X-Request-ID
X-Requested-With X-XSS-Protection X-Cache X-Cache-Hits X-Served-By
X-Timer Fastly-Debug-Digest Fastly-Client-IP Surrogate-Key
Surrogate-Control CDN-Loop CF-Connecting-IP True-Client-IP X-Real-IP
X-Correlation-ID X-Amzn-Trace-Id X-Api-Key X-Csrf-Token`

// Short English words (3-8 letters), from the mapbench word list.
const wordData = `deckel reaspire mether unrancid reseller khilat unpaunch gange prepink hereupon receipts jaw
mononymy practic ikona heptace parsec acinar slagger pontile matapan trompil nerval telepost
whorl unvain sodoku bludger tipsy cuffer rhyme sawder mantua quilt dagoba flicker nubbin lathe
orrery brisk tapioca sewage fusil gimlet harbor impala jostle kelpie lintel mosaic nutmeg ostler
pelican quoin rubble sorrel thistle umbra vellum wicker yonder zephyr amulet bobbin cobalt dingle
embers fathom gambit hamlet indigo jasper kettle lantern mallet napkin oyster pewter quiver ribbon`

const nkeys = 64 // power of two

func fields(s string) []string {
	var out []string
	start := -1
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ' ' || s[i] == '\n' {
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	return out
}

func strBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

type dataset struct {
	name string
	keys [][]byte
}

func intKeys(size int) dataset {
	rng := splitmix64(12345)
	buf := make([]byte, nkeys*size)
	keys := make([][]byte, nkeys)
	for i := range keys {
		k := buf[i*size : (i+1)*size]
		v := rng.next()
		for j := range k {
			k[j] = byte(v >> (8 * uint(j)))
		}
		keys[i] = k
	}
	return dataset{"int" + strconv.Itoa(size*8), keys}
}

func stringKeys(name string, ss []string, build func(string) string) dataset {
	keys := make([][]byte, nkeys)
	for i := range keys {
		s := ss[i%len(ss)]
		if build != nil {
			s = build(s)
		}
		keys[i] = strBytes(s)
	}
	return dataset{name, keys}
}

type hasher struct {
	name string
	f    func(p []byte, seed uint32) uint32
}

var hashers = []hasher{
	{"nop", func(p []byte, seed uint32) uint32 { return uint32(len(p)) + seed }},
	{"fnv", fnv32},
	{"fnvfin", fnvfin32},
	{"leveldb", leveldb32},
	{"tsip32", tsip32},
	{"tsip64", tsip64},
	{"sip13", sip13},
	{"sip24", sip24},
}

var sink uint32

func avgLen(keys [][]byte) int {
	n := 0
	for _, k := range keys {
		n += len(k)
	}
	return n / len(keys)
}

// measure returns the median ns/op over several runs, each lasting >= 200 ms.
func measure(f func(p []byte, seed uint32) uint32, keys [][]byte) (n int, nsPerOp float64) {
	loop := func(n int) {
		var s uint32
		for i := 0; i < n; i++ {
			s += f(keys[i&(nkeys-1)], uint32(i))
		}
		sink += s
	}
	n = 1
	if fixedN > 0 {
		n = fixedN
	} else {
		for {
			t := time.Now()
			loop(n)
			d := time.Since(t)
			if d >= 200*time.Millisecond || n >= 1<<26 {
				break
			}
			if d < time.Millisecond {
				n *= 10
			} else {
				n = n * int(250*time.Millisecond/d+1)
				if n < 2 {
					n = 2
				}
			}
		}
	}
	const reps = 5
	var ps [reps]int64
	for r := range ps {
		t := time.Now()
		loop(n)
		ps[r] = int64(time.Since(t)) * 1000 / int64(n) // ps/op
	}
	for i := 1; i < reps; i++ {
		for j := i; j > 0 && ps[j] < ps[j-1]; j-- {
			ps[j], ps[j-1] = ps[j-1], ps[j]
		}
	}
	return n, float64(ps[reps/2]) / 1000
}

func f1(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

func runDataset(ds dataset) {
	println("--", ds.name, "avg key bytes:", avgLen(ds.keys))
	var nop float64
	for _, h := range hashers {
		n, ns := measure(h.f, ds.keys)
		if h.name == "nop" {
			nop = ns
		}
		net := ns - nop
		line := h.name + " n=" + strconv.Itoa(n) + " ns/op=" + f1(ns) + " net_ns/op=" + f1(net)
		if cpuMHz > 0 {
			line += " net_cycles/op=" + f1(net*float64(cpuMHz)/1000)
		}
		println(line)
	}
}

func main() {
	// Give a USB-serial host time to attach.
	time.Sleep(3 * time.Second)
	for {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		println("== hash benchmark; cpu MHz:", cpuMHz, "heap sys/inuse:", ms.HeapSys, ms.HeapInuse)
		// Ints first.
		runDataset(intKeys(4))
		runDataset(intKeys(8))
		// Then strings.
		runDataset(stringKeys("words", fields(wordData), nil))
		runDataset(stringKeys("headers", fields(headerData), nil))
		runDataset(stringKeys("long40", fields(wordData), func(w string) string {
			return "this-is-a-longer-key-for-testing-" + w
		}))
		long := make([]byte, 128)
		for i := range long {
			long[i] = byte(i*7 + 1)
		}
		bigs := make([][]byte, nkeys)
		for i := range bigs {
			bigs[i] = long
		}
		runDataset(dataset{"big128", bigs})
		println("== DONE")
		time.Sleep(10 * time.Second)
	}
}
