package httpx

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func WithRetry(retries int) func(Client) Client {
	return func(c Client) Client {
		return ClientFunc(func(req *http.Request) (resp *http.Response, err error) {
			if req.Body == nil {
				req.Body = ioutil.NopCloser(bytes.NewReader(nil))
			}

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return
			}

			for attempt := 0; attempt < retries; attempt++ {
				req.Body = ioutil.NopCloser(bytes.NewReader(body))

				resp, err = c.Do(req)
				if err != nil {
					log.Printf("An error while making request to Eolymp API: %v", err)
					return nil, err
				}

				if err != nil || resp.StatusCode != http.StatusOK {
					body, _ = ioutil.ReadAll(resp.Body)
					log.Printf("Server returned an error, status code %d: %s", resp.StatusCode, body)
					time.Sleep(time.Second * time.Duration((attempt+1)*(attempt+1)))
					continue
				}

				return resp, nil
			}

			return resp, err
		})
	}
}
