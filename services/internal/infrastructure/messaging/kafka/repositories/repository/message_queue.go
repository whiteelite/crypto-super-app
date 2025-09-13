package repository

import (
	"context"
	"errors"
	"sync"

	sdk "github.com/segmentio/kafka-go"
	domainrepos "github.com/whiteelite/superapp/internal/domain/repositories"
	shared "github.com/whiteelite/superapp/pkg/shared/domain/entities"
)

// KafkaMessageQueueParams implements repositories.MessageQueueParams
// and provides configuration for initializing KafkaMessageQueue.
type KafkaMessageQueueParams struct {
	// Required
	Brokers []string
	Topic   string

	// Optional
	GroupID          string
	ToProduceBufSize int
	ToConsumeBufSize int
}

func (p KafkaMessageQueueParams) Get() map[string]any {
	return map[string]any{
		"brokers":         p.Brokers,
		"topic":           p.Topic,
		"groupId":         p.GroupID,
		"toProduceBuffer": p.ToProduceBufSize,
		"toConsumeBuffer": p.ToConsumeBufSize,
	}
}

// KafkaMessageQueue implements domain MessageQueue interfaces
// by bridging to the existing StartProducer/StartConsumer workers.
type KafkaMessageQueue struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	reader *sdk.Reader
	writer *sdk.Writer

	// External facing channels (Entity based)
	toProduce chan shared.Entity
	toConsume chan shared.Entity

	// Internal bridges (generic pointer channels)
	prodBucket    chan *shared.Entity
	consBucket    chan *shared.Entity
	errorsProd    chan error
	errorsCons    chan error
	confirmations chan *shared.Entity
}

// InitializeKafkaMessageQueue creates a KafkaMessageQueue using params.
func InitializeKafkaMessageQueue(params domainrepos.MessageQueueParams) domainrepos.MessageQueue {
	typed, _ := params.(KafkaMessageQueueParams)

	// defaults
	if typed.ToProduceBufSize <= 0 {
		typed.ToProduceBufSize = 1024
	}
	if typed.ToConsumeBufSize <= 0 {
		typed.ToConsumeBufSize = 1024
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Writer
	writer := &sdk.Writer{
		Addr:         sdk.TCP(typed.Brokers...),
		Topic:        typed.Topic,
		RequiredAcks: sdk.RequireAll,
		Balancer:     &sdk.LeastBytes{},
	}

	// Reader
	reader := sdk.NewReader(sdk.ReaderConfig{
		Brokers: typed.Brokers,
		Topic:   typed.Topic,
		GroupID: typed.GroupID,
	})

	mq := &KafkaMessageQueue{
		ctx:           ctx,
		cancel:        cancel,
		wg:            wg,
		reader:        reader,
		writer:        writer,
		toProduce:     make(chan shared.Entity, typed.ToProduceBufSize),
		toConsume:     make(chan shared.Entity, typed.ToConsumeBufSize),
		prodBucket:    make(chan *shared.Entity, typed.ToProduceBufSize),
		consBucket:    make(chan *shared.Entity, typed.ToConsumeBufSize),
		errorsProd:    make(chan error, 16),
		errorsCons:    make(chan error, 16),
		confirmations: make(chan *shared.Entity, 16),
	}

	mq.startWorkers()
	return mq
}

func (q *KafkaMessageQueue) startWorkers() {
	// Producer worker uses prodBucket
	q.wg.Add(1)
	go StartProducer[shared.Entity](q.ctx, q.wg, q.writer, q.prodBucket, q.errorsProd)

	// Consumer worker fills consBucket and uses confirmations to commit
	q.wg.Add(1)
	go StartConsumer[shared.Entity](q.ctx, q.wg, q.reader, q.consBucket, q.errorsCons, q.confirmations)

	// Bridge external toProduce -> prodBucket (*Entity)
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-q.ctx.Done():
				return
			case e, ok := <-q.toProduce:
				if !ok {
					return
				}
				// allocate a new variable to take address
				entity := e
				q.prodBucket <- &entity
			}
		}
	}()

	// Bridge consBucket (*Entity) -> toConsume and confirmation
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case <-q.ctx.Done():
				return
			case ptr, ok := <-q.consBucket:
				if !ok {
					return
				}
				if ptr == nil {
					continue
				}
				q.toConsume <- *ptr
				// Send confirmation pointer back for commit
				q.confirmations <- ptr
			}
		}
	}()
}

// ToConsumeBuffered exposes the consumer channel of entities.
func (q *KafkaMessageQueue) ToConsumeBuffered() <-chan shared.Entity {
	return q.toConsume
}

// ToProduceBuffered exposes the producer channel of entities.
func (q *KafkaMessageQueue) ToProduceBuffered() chan<- shared.Entity {
	return q.toProduce
}

// Close stops workers and closes resources.
func (q *KafkaMessageQueue) Close() {
	// First cancel context so workers stop accepting work
	if q.cancel != nil {
		q.cancel()
	}

	// Close external producer channel
	// Avoid panics: close only once and only if not nil
	var once sync.Once
	once.Do(func() {
		if q.toProduce != nil {
			close(q.toProduce)
		}
	})

	// Close internal producer bucket to unblock StartProducer
	if q.prodBucket != nil {
		close(q.prodBucket)
	}

	// Close reader/writer
	if q.reader != nil {
		_ = q.reader.Close()
	}
	if q.writer != nil {
		_ = q.writer.Close()
	}

	// Wait for all goroutines to finish
	q.wg.Wait()

	// Now it is safe to close consumer-facing channel
	if q.toConsume != nil {
		close(q.toConsume)
	}
}

// Compile-time assertions to ensure interface conformance
var _ domainrepos.MessageQueueConsumer = (*KafkaMessageQueue)(nil)
var _ domainrepos.MessageQueueProducer = (*KafkaMessageQueue)(nil)
var _ domainrepos.MessageQueue = (*KafkaMessageQueue)(nil)

// Helper to ensure required params are set.
func ValidateKafkaParams(p KafkaMessageQueueParams) error {
	if len(p.Brokers) == 0 {
		return errors.New("kafka brokers are required")
	}
	if p.Topic == "" {
		return errors.New("kafka topic is required")
	}
	return nil
}
