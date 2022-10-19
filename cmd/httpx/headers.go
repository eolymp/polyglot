package httpx

import "net/http"

// WithHeaders for outgoing HTTP calls
func WithHeaders(headers map[string][]string) func(Client) Client {
	return func(c Client) Client {
		return ClientFunc(func(req *http.Request) (*http.Response, error) {
			for h, v := range headers {
				req.Header[h] = v
			}

			return c.Do(req)
		})
	}
}
