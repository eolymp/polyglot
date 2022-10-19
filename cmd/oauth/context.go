package oauth

import "context"

type contextKey int

const contextToken contextKey = iota

func TokenFromContext(ctx context.Context) (*Token, bool) {
	token, ok := ctx.Value(contextToken).(*Token)
	return token, ok
}

func IdentityFromContext(ctx context.Context) (*Identity, bool) {
	token, ok := TokenFromContext(ctx)
	if !ok {
		return nil, false
	}

	return token.Identity, token.Identity != nil
}

func ContextWithToken(ctx context.Context, token *Token) context.Context {
	return context.WithValue(ctx, contextToken, token)
}
