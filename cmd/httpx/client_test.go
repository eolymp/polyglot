package httpx_test

import (
	"fmt"
	"github.com/eolymp/polyglot/cmd/httpx"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func ExampleNewClient() {
	// build HTTPx client
	client := httpx.NewClient(
		// provide base HTTP client
		&http.Client{Timeout: 30 * time.Second},
		// add additional headers
		httpx.WithHeaders(map[string][]string{
			"User-Agent": {"Go HTTPx client"},
		}),
	)

	// build request
	req, err := http.NewRequest(http.MethodGet, "https://magento.com", nil)
	if err != nil {
		panic(err)
	}

	// send request
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("non-200 response, got %v", resp.StatusCode))
	}

	// process response
}

func TestNewClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		//nolint:errcheck
		fmt.Fprintln(rw, "hello world")
	}))
	defer srv.Close()

	cli := httpx.NewClient(&http.Client{Timeout: time.Second})

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal("Request can not be created:", err)
	}

	resp, err := cli.Do(req)
	if err != nil {
		t.Fatal("Request to test server has failed:", err)
	}

	//nolint:errcheck
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Response status code is not 200, got %v instead", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read response body:", err)
	}

	got := string(body)

	if want := "hello world\n"; got != want {
		t.Errorf("Response is not correct: want %v, got %v", want, got)
	}
}
