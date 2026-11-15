package mapbench

import "strings"

// Keys are irregular on purpose. Dense sequential keys flatter FNV.

// splitmix64 gives the same datasets on every target.
type splitmix64 uint64

func (s *splitmix64) next() uint64 {
	*s += 0x9e3779b97f4a7c15
	z := uint64(*s)
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	return z ^ z>>31
}

// randInts returns n distinct pseudo-random ints.
func randInts(n int) []int {
	rng := splitmix64(12345)
	seen := make(map[int]struct{}, n)
	out := make([]int, 0, n)
	for len(out) < n {
		v := int(rng.next())
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

var words []string

// wordKeys returns n distinct short (3-8 letter) English words.
func wordKeys(n int) []string {
	if words == nil {
		words = strings.Fields(wordData)
	}
	return words[:n]
}

// Real header names, in canonical case.
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

var headers []string

// headerKeys returns n distinct header names. The first 230 or so are real
// and the rest are synthesized as x-word-word.
func headerKeys(n int) []string {
	if headers == nil {
		real := strings.Fields(headerData)
		headers = append(headers, real...)
		for _, h := range real {
			headers = append(headers, strings.ToLower(h))
		}
		rng := splitmix64(777)
		ws := wordKeys(16384)
		seen := make(map[string]struct{}, 20000)
		for _, h := range headers {
			seen[h] = struct{}{}
		}
		for len(headers) < 16384 {
			h := "x-" + ws[rng.next()%uint64(len(ws))] + "-" + ws[rng.next()%uint64(len(ws))]
			if _, dup := seen[h]; dup {
				continue
			}
			seen[h] = struct{}{}
			headers = append(headers, h)
		}
	}
	return headers[:n]
}
