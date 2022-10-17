package httpx_test

import (
	"context"
	"github.com/eolymp/go-packages/httpx"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type CredentialsFunc func(context.Context) (string, error)

func (f CredentialsFunc) Credentials(ctx context.Context) (string, error) {
	return f(ctx)
}

func TestWithCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") == "Bearer Token" {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cli := httpx.NewClient(
		&http.Client{Timeout: time.Second},
		httpx.WithCredentials(CredentialsFunc(func(ctx context.Context) (string, error) {
			return "Bearer Token", nil
		})),
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
