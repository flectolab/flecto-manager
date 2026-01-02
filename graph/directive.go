package graph

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/flectolab/flecto-manager/auth"
)

var ErrUnauthorized = errors.New("unauthorized")

func PublicDirective(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	ctx = context.WithValue(ctx, "public", true)
	return next(ctx)
}

func AuthMiddleware(ctx context.Context, next graphql.Resolver) (any, error) {
	fc := graphql.GetFieldContext(ctx)

	if fc != nil {
		for _, d := range fc.Field.Directives {
			if d.Name == "public" {
				return next(ctx)
			}
		}
	}

	if auth.GetUser(ctx) == nil {
		return nil, ErrUnauthorized
	}
	return next(ctx)
}
