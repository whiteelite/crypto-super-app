package repositories

import (
	"context"

	shared "github.com/whiteelite/steppe/pkg/shared/domain/entities"
)

type Pagination interface {
	Limit() int64
	Offset() int64
}

type MappedModel map[string]any

type UpdateMapped[T shared.Entity] interface {
	UpdateMapped(ctx context.Context, entity T, model *MappedModel) error
}

type CRUD[T shared.Entity, P Pagination] interface {
	Create(ctx context.Context, entitiy T) error
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, entity T) error
	Paginate(ctx context.Context, entitiy P) ([]T, error)
	UpdateMapped() UpdateMapped[T]
}

type DefaultDatabaseRepository interface {
	CRUD[shared.Entity, Pagination]
}

type MessageQueueParams interface {
	Get() map[string]any
}

type InitializeMessageQueue func(MessageQueueParams) MessageQueue

type MessageQueueConsumer interface {
	ToConsumeBuffered() <-chan shared.Entity
	Close()
}

type MessageQueueProducer interface {
	ToProduceBuffered() chan<- shared.Entity
	Close()
}

type MessageQueue interface {
	MessageQueueProducer
	MessageQueueConsumer
}
