package repository

import (
	"context"
	"sync"

	json "github.com/goccy/go-json"

	sdk "github.com/segmentio/kafka-go"
	mapper "github.com/whiteelite/superapp/internal/infrastructure/messaging/kafka/repositories/mapper"
	shared "github.com/whiteelite/superapp/pkg/shared/domain/entities"
)

func StartProducer[T any | shared.Entity](
	ctx context.Context,
	wg *sync.WaitGroup,
	writer *sdk.Writer,
	bucket <-chan *T,
	errors chan<- error,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			close(errors)
			return
		default:
			request := <-bucket

			model, err := mapper.ToMessage(request)
			if err != nil {
				errors <- err
				continue
			}

			serialized, err := json.Marshal(model)
			if err != nil {
				errors <- err
				continue
			}
			err = writer.WriteMessages(ctx, sdk.Message{
				Key:   []byte(model.Hash),
				Value: []byte(serialized),
			})
			if err != nil {
				errors <- err
				continue
			}
		}
	}

}
