package httpx_test

import (
	"github.com/eolymp/polyglot/cmd/httpx"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWithHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Test") != "FooBarBaz" {
			t.Error("Request does not contain X-Test header")
		}

		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cli := httpx.NewClient(
		&http.Client{Timeout: time.Second},
		httpx.WithHeaders(map[string][]string{
			"X-Test": {"FooBarBaz"},
		}),
	)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal("Request can not be created:", err)
	}

	resp, err := cli.Do(req)
	if err != nil {
		t.Fatal("Request to test server has failed:", err)
	}

	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Response status code is not 200, got %v instead", resp.StatusCode)
	}
}
