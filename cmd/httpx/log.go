package httpx

import (
	"fmt"
	"net/http"
	"time"
)

type logger interface {
	Error(msg string, data map[string]interface{})
	Debug(msg string, data map[string]interface{})
}

// Log middleware logs every request to the server
func Log(log logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			hijack, _ := rw.(http.Hijacker)

			writer := &responseWriter{
				ResponseWriter: rw,
				Hijacker:       hijack,
				status:         http.StatusOK,
			}

			start := time.Now()
			method := req.Method
			path := req.URL.Path

			defer func() {
				latency := time.Since(start)
				status := writer.status

				data := map[string]interface{}{
					"http_method":    req.Method,
					"http_url":       req.URL.String(),
					"http_status":    status,
					"http_useragent": req.UserAgent(),
					"http_referer":   req.Referer(),
					"http_latency":   latency,
					"http_protocol":  req.Proto,
					"context":        req.Context(),
				}

				if status/100 == 5 {
					log.Error(fmt.Sprintf("%s request to %s failed with status code %3d", method, path, status), data)
				} else {
					log.Debug(fmt.Sprintf("%s request to %s with status code %3d", method, path, status), data)
				}
			}()

			h.ServeHTTP(writer, req)
		})
	}
}
