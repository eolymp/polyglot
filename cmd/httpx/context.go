package httpx

type contextKey int

const (
	contextLanguageTag contextKey = iota
	contextRequestHeaders
	contextRemoteAddr
)
