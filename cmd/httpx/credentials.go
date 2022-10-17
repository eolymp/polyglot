package httpx

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"sync/atomic"
)

type credentials interface {
	Credentials(context.Context) (string, error)
}

// WithCredentials provides authentication info for outgoing HTTP calls
// In case server replies 401 this middleware fetches credentials using credential provider (eq. OAuth) and sets Authentication
// header. It re-uses credentials for subsequent calls until server replies 401 again, in which case it will try to fetch new
// credentials using credential provider.
func WithCredentials(cred credentials) func(Client) Client {
	var auth atomic.Value

	return func(c Client) Client {
		if cred == nil {
			return c
		}

		return ClientFunc(func(req *http.Request) (resp *http.Response, err error) {
			if req.Body == nil {
				req.Body = ioutil.NopCloser(bytes.NewReader(nil))
			}

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return
			}

			for attempt := 0; attempt < 2; attempt++ {
				req.Body = ioutil.NopCloser(bytes.NewReader(body))

				if header, ok := auth.Load().(string); ok && header != "" {
					req.Header.Set("Authorization", header)
				}

				resp, err = c.Do(req)
				if err != nil {
					return
				}

				if resp.StatusCode != http.StatusUnauthorized {
					return
				}

				// 401 free resources and retry
				resp.Body.Close()

				// fetch new authentication header
				var header string

				header, err = cred.Credentials(req.Context())
				if err != nil {
					return
				}

				auth.Store(header)
			}

			return
		})
	}
}
