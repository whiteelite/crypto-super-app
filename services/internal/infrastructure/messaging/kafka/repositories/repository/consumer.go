package repository

import (
	"context"
	"sync"

	json "github.com/goccy/go-json"

	sdk "github.com/segmentio/kafka-go"
	mapper "github.com/whiteelite/superapp/internal/infrastructure/messaging/kafka/repositories/mapper"
	models "github.com/whiteelite/superapp/internal/infrastructure/messaging/kafka/repositories/models"
	shared "github.com/whiteelite/superapp/pkg/shared/domain/entities"
)

func StartConsumer[T any | shared.Entity](
	ctx context.Context,
	wg *sync.WaitGroup,
	reader *sdk.Reader,
	bucket chan<- *T,
	errors chan<- error,
	confirmed <-chan *T,
) {
	defer wg.Done()

	// Read messages from the reader
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				data, err := reader.ReadMessage(ctx)
				if err != nil {
					errors <- err
					continue
				}

				model := new(models.Message)
				if err := json.Unmarshal(data.Value, &model); err != nil {
					errors <- err
					continue
				}

				message, err := mapper.FromMessage[T](model)
				if err != nil {
					errors <- err
					continue
				}

				bucket <- message
			}
		}
	}()

	// Confirm messages
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return

			case data, ok := <-confirmed:
				if !ok {
					continue
				}

				request := data

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

				reader.CommitMessages(
					ctx,
					sdk.Message{
						Key:   []byte(model.Hash),
						Value: serialized,
					},
				)
			}
		}
	}()

	<-ctx.Done()
	close(bucket)
	close(errors)
}
