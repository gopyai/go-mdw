package mdw2

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	sErrAuth          = "Auth-Key does not match"
	sErrMethod        = "Unexpected HTTP Method"
	sErrTimeout       = "Request timeout"
	sErrOpenOverLimit = "Too many opened connection"
	sErrTPSOverLimit  = "Request denied due to limited TPS"

	errTPSLimitValue  = errors.New("Valid limit value is: limit=0 or limit >= 1")
	errOpenLimitValue = errors.New("Valid limit value must is limit >= 0")
)

func MustMethod(must string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				if r.Method != must {
					http.Error(w, sErrMethod, http.StatusBadRequest)
					return
				}
				next.ServeHTTP(w, r)

			})

	}
}

func TimeLimit(limit time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, limit, sErrTimeout)
	}
}

func RequestSizeLimit(size int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				r.Body = http.MaxBytesReader(w, r.Body, size)
				next.ServeHTTP(w, r)

			})

	}
}

func StripPrefix(prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				if strings.HasPrefix(r.RequestURI, prefix) {
					r.RequestURI = r.RequestURI[len(prefix):]
				}
				next.ServeHTTP(w, r)

			})

	}
}

func AuthKeys(keys []string) func(http.Handler) http.Handler {
	kmap := make(map[string]bool)
	for _, k := range keys {
		kmap[k] = true
	}

	var errStr func(http.Header) string
	if len(kmap) == 0 {
		errStr = func(h http.Header) string { return "" }
	} else {
		errStr = func(h http.Header) string {
			key := h.Get("Auth-Key")
			if len(key) == 0 {
				return sErrAuth
			}
			hash := SHA256Hash(key)
			if !kmap[hash] {
				return sErrAuth
			}
			return ""
		}
	}

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				if errStr := errStr(r.Header); len(errStr) != 0 {
					http.Error(w, errStr, http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)

			})

	}
}

func OpenLimit(limit int) func(http.Handler) http.Handler {
	if limit < 0 {
		panic(errOpenLimitValue)
	}
	var lck struct {
		sync.Mutex
		cnt int
	}

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				// Increment number of connection
				lck.Lock()
				if lck.cnt >= limit {
					lck.Unlock()
					http.Error(w, sErrOpenOverLimit, http.StatusTooManyRequests)
					return
				}
				lck.cnt++
				lck.Unlock()

				// Next
				next.ServeHTTP(w, r)

				// Decrement number of connection
				lck.Lock()
				lck.cnt--
				lck.Unlock()

			})

	}
}

func TPSLimit(limit, burst float64) func(http.Handler) http.Handler {
	if limit != 0 && limit < 1 {
		panic(errTPSLimitValue)
	}
	if burst < 1 {
		panic(errTPSLimitValue)
	}

	var lck struct {
		sync.Mutex
		tx float64
		t  time.Time
	}
	lck.t = time.Now()

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				lck.Lock()

				// Calculate available tx counter
				t := time.Now()
				secs := t.Sub(lck.t).Seconds()
				lck.tx += secs * limit
				if lck.tx > burst {
					lck.tx = burst
				}
				lck.t = t

				if lck.tx < 1 {
					// Not enough counter, error
					lck.Unlock()
					http.Error(w, sErrTPSOverLimit, http.StatusTooManyRequests)
					return
				}

				// Consume the counter
				lck.tx--
				lck.Unlock()

				// Next
				next.ServeHTTP(w, r)

			})

	}
}

func SHA256Hash(msg string) string {
	h := sha256.New()
	h.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
