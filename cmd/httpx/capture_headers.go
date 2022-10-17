package httpx

import (
	"context"
	"net"
	"net/http"
	"strings"
)

var private = networks([]string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"fc00::/7",
})

func networks(cidrs []string) (nets []net.IPNet) {
	for _, cidr := range cidrs {
		if _, network, err := net.ParseCIDR(strings.TrimSpace(cidr)); err == nil {
			nets = append(nets, *network)
		}
	}

	return
}

func isPublicIP(ip net.IP) bool {
	if !ip.IsGlobalUnicast() {
		return false
	}

	for _, p := range private {
		if p.Contains(ip) {
			return false
		}
	}

	return true
}

func CaptureHeaders() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, contextRequestHeaders, r.Header)
			ctx = context.WithValue(ctx, contextRemoteAddr, r.RemoteAddr)

			r = r.WithContext(ctx)
			h.ServeHTTP(w, r)
		})
	}
}

func RequestHeaders(ctx context.Context) http.Header {
	if header, ok := ctx.Value(contextRequestHeaders).(http.Header); ok && header != nil {
		return header
	}

	return make(map[string][]string)
}

func ClientIP(ctx context.Context) (ips []string) {
	// get all proxies
	xff := RequestHeaders(ctx).Get("X-Forwarded-For")

	for _, ip := range strings.Split(xff, ",") {
		ip = strings.TrimSpace(ip)
		if pip := net.ParseIP(ip); pip != nil && isPublicIP(pip) {
			ips = append(ips, ip)
		}
	}

	// get remote addr
	if addr, ok := ctx.Value(contextRemoteAddr).(string); ok {
		if host, _, err := net.SplitHostPort(addr); err == nil {
			if pip := net.ParseIP(host); pip != nil && isPublicIP(pip) {
				ips = append(ips, host)
			}
		}
	}

	return ips
}
