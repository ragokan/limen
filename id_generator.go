package limen

import (
	"context"

	"github.com/google/uuid"
)

type uuidv7IDGenerator struct{}

// NewUUIDv7IDGenerator returns an ID generator that creates UUIDv7 string IDs.
func NewUUIDv7IDGenerator() IDGenerator {
	return uuidv7IDGenerator{}
}

func (uuidv7IDGenerator) Generate(ctx context.Context) (any, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return id.String(), nil
}

func (uuidv7IDGenerator) GetColumnType() ColumnType {
	return ColumnTypeUUID
}
