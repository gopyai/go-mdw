package mdw2_test

import (
	"bytes"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestMustMethod_fail(t *testing.T) {
	b, s, e := doHttpPost(nil, "http://localhost:8080/MustGet", []byte("Hello"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("Unexpected HTTP Method\n")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusServiceUnavailable
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestMustMethod_ok(t *testing.T) {
	b, s, e := doHttpPost(nil, "http://localhost:8080/MustPost", []byte("Hello"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("URI: /MustPost is OK")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusOK
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestTimeLimit(t *testing.T) {
	b, s, e := doHttpPost(nil, "http://localhost:8080/TimeLimit", []byte("Hello"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("Request timeout")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusServiceUnavailable
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestRequestSizeLimit_ok(t *testing.T) {
	b, s, e := doHttpPost(nil, "http://localhost:8080/RequestSizeLimit", []byte("Hello"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("URI: /RequestSizeLimit is OK")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusOK
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestRequestSizeLimit_fail(t *testing.T) {
	b, s, e := doHttpPost(nil, "http://localhost:8080/RequestSizeLimit", []byte("This is a long string"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusOK
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestStrip(t *testing.T) {
	b, s, e := doHttpPost(nil, "http://localhost:8080/Strip/Hello", []byte("Hello"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("URI: /Hello is OK")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusOK
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestAuth_ok(t *testing.T) {
	b, s, e := doHttpPost(map[string]string{
		"Auth-Key": "secret",
	}, "http://localhost:8080/Auth", []byte("Hello"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("URI: /Auth is OK")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusOK
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestAuth_fail(t *testing.T) {
	b, s, e := doHttpPost(map[string]string{
		"Auth-Key": "fail",
	}, "http://localhost:8080/Auth", []byte("Hello"))

	if e != nil {
		t.Error("Error:", e)
	}

	wb := []byte("Auth-Key does not match\n")
	if bytes.Compare(b, wb) != 0 {
		t.Errorf("HTTP response:\nGot:\n%s\nWant:\n%s\n", b, wb)
	}

	ws := http.StatusServiceUnavailable
	if s != ws {
		t.Errorf("HTTP status, got: %d, want: %d", s, ws)
	}
}

func TestOpenLimit(t *testing.T) {
	var lck sync.Mutex
	ok := 0

	var wg sync.WaitGroup
	for i := 0; i < OPEN_LIMIT_TEST_TOTAL; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, s, e := doHttpPost(nil, "http://localhost:8080/OpenLimit", []byte("Hello"))

			if e != nil {
				t.Error("Error:", e)
			}

			lck.Lock()
			defer lck.Unlock()
			if s == http.StatusOK && bytes.Compare(b, []byte("OK")) == 0 {
				ok++
				//				t.Log("OK")
			} else if s == http.StatusServiceUnavailable && bytes.Compare(b, []byte("Too many opened connection\n")) == 0 {
				//				t.Log("Fail")
			} else {
				t.Errorf("Unknown result: Status code=%d, response=%s", s, string(b))
			}
		}()
		time.Sleep(time.Millisecond * 10)
	}
	wg.Wait()

	wok := OPEN_LIMIT * 2
	if ok != wok {
		t.Errorf("Unknown result:\nok=%d, want=%d", ok, wok)
	}
}

func TestTPSLimit(t *testing.T) {
	var wg sync.WaitGroup

	// Will test for TPS_TEST_SECS seconds

	// Clear all TPS internal counter by calling HTTP
	for i := 0; i < TPS_LIMIT; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doHttpPost(nil, "http://localhost:8080/TPSLimit", []byte("Hello"))
		}()
	}
	wg.Wait()

	// Start the timer
	t0 := time.Now()

	// Initiate lock and counter
	var lck sync.Mutex
	ok := 0

	// Measure TPS
	for {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, s, e := doHttpPost(nil, "http://localhost:8080/TPSLimit", []byte("Hello"))

			if e != nil {
				t.Error("Error:", e)
			}

			lck.Lock()
			defer lck.Unlock()
			if s == http.StatusOK && bytes.Compare(b, []byte("URI: /TPSLimit is OK")) == 0 {
				ok++
			} else if s == http.StatusServiceUnavailable && bytes.Compare(b, []byte("Request denied due to limited TPS\n")) == 0 {
			} else {
				t.Errorf("Unknown result: Status code=%d, response=%s", s, string(b))
			}
		}()
		time.Sleep(time.Millisecond)

		// Test for TPS_SECS seconds
		if time.Since(t0).Seconds() >= TPS_TEST_SECS {
			break
		}
	}
	wg.Wait()

	secs := time.Since(t0).Seconds()
	tps := float64(ok) / secs

	var wtps float64 = TPS_LIMIT + 1
	if tps > wtps {
		t.Errorf("Unknown result:\ntps=%f, want=%f", tps, wtps)
	}
}
