package templ

import "context"

type nonceKey struct{}

// WithNonce returns a new context with the CSP nonce.
func WithNonce(ctx context.Context, nonce string) context.Context {
	return context.WithValue(ctx, nonceKey{}, nonce)
}

// GetNonce returns the CSP nonce from the context.
func GetNonce(ctx context.Context) string {
	if nonce, ok := ctx.Value(nonceKey{}).(string); ok {
		return nonce
	}
	return ""
}
