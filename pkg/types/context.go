package types

import "context"

type Context struct {
	context.Context
}

type ContextOption func(ctx *Context)

func NewContext(options ...ContextOption) *Context {
	ctx := &Context{}

	for _, opt := range options {
		opt(ctx)
	}

	return ctx
}

func WithBase(base context.Context) ContextOption {
	return func(ctx *Context) {
		ctx.Context = base
	}
}
