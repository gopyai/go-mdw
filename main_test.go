package mdw2_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gopyai/go-mdw"
	"github.com/justinas/alice"
)

const (
	OPEN_LIMIT            = 3
	OPEN_LIMIT_TEST_TOTAL = 13

	TPS_LIMIT     = 5
	TPS_BURST     = 1.5
	TPS_TEST_SECS = 2
)

func TestMain(m *testing.M) {
	mux := http.NewServeMux()

	mux.Handle("/MustGet", alice.New(
		mdw2.MustMethod("GET"),
	).Then(http.HandlerFunc(handler)))

	mux.Handle("/MustPost", alice.New(
		mdw2.MustMethod("POST"),
	).Then(http.HandlerFunc(handler)))

	mux.Handle("/TimeLimit", alice.New(
		mdw2.TimeLimit(time.Millisecond * 10),
	).Then(http.HandlerFunc(delayedHandler)))

	mux.Handle("/RequestSizeLimit", alice.New(
		mdw2.RequestSizeLimit(10),
	).Then(http.HandlerFunc(handler)))

	mux.Handle("/OpenLimit", alice.New(
		mdw2.OpenLimit(OPEN_LIMIT),
	).Then(http.HandlerFunc(delayedHandler)))

	mux.Handle("/TPSLimit", alice.New(
		mdw2.TPSLimit(TPS_LIMIT, TPS_BURST),
	).Then(http.HandlerFunc(handler)))

	mux.Handle("/Strip/", alice.New(
		mdw2.StripPrefix("/Strip"),
	).Then(http.HandlerFunc(handler)))

	mux.Handle("/Auth", alice.New(
		mdw2.AuthKeys([]string{mdw2.SHA256Hash("secret"), mdw2.SHA256Hash("key")}),
	).Then(http.HandlerFunc(handler)))

	go func() {
		if e := http.ListenAndServe(":8080", mux); e != nil {
			fmt.Println("Error:", e)
			os.Exit(1)
		}
	}()
	time.Sleep(time.Millisecond * 100)
	os.Exit(m.Run())
}

func handler(w http.ResponseWriter, r *http.Request) {
	_, e := ioutil.ReadAll(r.Body)
	if e != nil {
		// fmt.Println("Server Error:", e)
		return
	}

	_, e = w.Write([]byte(fmt.Sprintf("URI: %s is OK", r.RequestURI)))
	panicIf(e)
}

func delayedHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Millisecond * 100)
	_, e := ioutil.ReadAll(r.Body)
	if e != nil {
		// fmt.Println("Server Error:", e)
		return
	}
	_, e = w.Write([]byte("OK"))
	panicIf(e)
}

func doHttpPost(header map[string]string, url string, body []byte) ([]byte, int, error) {
	cli := new(http.Client)

	req, e := http.NewRequest("POST", url, bytes.NewReader(body))
	if e != nil {
		return nil, 0, e
	}

	for k, v := range header {
		req.Header.Set(k, v)
	}

	res, e := cli.Do(req)
	if e != nil {
		return nil, 0, e
	}
	defer res.Body.Close()

	b, e := ioutil.ReadAll(res.Body)
	return b, res.StatusCode, e
}

func panicIf(e error) {
	if e != nil {
		panic(e)
	}
}
