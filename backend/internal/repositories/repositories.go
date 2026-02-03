package repositories

import (
	"context"
	"database/sql"
	"errors"
)

func fetchAndMap[T any, R any](
	ctx context.Context,
	queryFunc func(context.Context) (T, error),
	mapFunc func(T) R,
) (R, error) {
	var zero R

	res, err := queryFunc(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return zero, nil
		}
		return zero, err
	}

	return mapFunc(res), nil
}

func fetchAndMapSlice[T any, R any](
	ctx context.Context,
	queryFunc func(context.Context) ([]T, error),
	mapFunc func(T) R,
) ([]R, error) {
	rows, err := queryFunc(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]R, len(rows))
	for i, v := range rows {
		result[i] = mapFunc(v)
	}
	return result, nil
}
