package domain

import "context"

type Repository interface {
	GetAll(ctx context.Context) (map[Key]Value, error)
	GetByKey(ctx context.Context, key Key) (Value, error)
}
