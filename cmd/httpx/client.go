package httpx

import (
	"net/http"
)

// Client is well-known HTTP client interface
type Client interface {
	Do(*http.Request) (*http.Response, error)
}

// NewClient with middleware
func NewClient(cli Client, mw ...func(Client) Client) Client {
	for _, m := range mw {
		cli = m(cli)
	}

	return cli
}

// ClientFunc which implements Client interface
type ClientFunc func(req *http.Request) (*http.Response, error)

// Do HTTP request
func (f ClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}
