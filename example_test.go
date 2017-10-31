package mdw2_test

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gopyai/mdw2"
	"github.com/justinas/alice"
)

func Example() {
	mux := http.NewServeMux()

	// Different path may have different middleware chains
	mux.Handle("/path1/", alice.New(
		mdw2.MustMethod("POST"),                            // Must use POST method
		mdw2.AuthKeys([]string{mdw2.SHA256Hash("secret")}), // Allowed keys for Auth-Key header
		mdw2.RequestSizeLimit(10000),                       // Max request size = 10Kbytes
		mdw2.TimeLimit(time.Second*10),                     // Will timeout if elapsed time is more than 10 seconds
		mdw2.OpenLimit(10),                                 // Max open connection = 10
		mdw2.TPSLimit(100, 10),                             // Max TPS = 100, burst up to 10 calls
		mdw2.StripPrefix("/path1"),
	).Then(http.HandlerFunc(handler)))

	mux.Handle("/path2/", alice.New(
		mdw2.TimeLimit(time.Second),
	).Then(http.HandlerFunc(handler)))

	go func() {
		if e := http.ListenAndServe(":8080", mux); e != nil {
			fmt.Println("Error:", e)
			os.Exit(1)
		}
	}()
}
