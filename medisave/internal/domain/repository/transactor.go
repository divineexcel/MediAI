package repository

import "context"

type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type txContextKey struct{}

func WithTransaction(ctx context.Context, tx interface{}) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func GetTransaction(ctx context.Context) interface{} {
	return ctx.Value(txContextKey{})
}
