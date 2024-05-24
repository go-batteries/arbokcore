package supervisors

import (
	"arbokcore/pkg/queuer"
	"context"

	"github.com/rs/zerolog/log"
)

type MetadataUpdateConsumerProducer struct {
	queue  queuer.Queuer
	demand chan int
}

func NewMetadataNotifier(queue queuer.Queuer) *MetadataUpdateConsumerProducer {
	return &MetadataUpdateConsumerProducer{
		queue:  queue,
		demand: make(chan int, 1),
	}
}

func (slf *MetadataUpdateConsumerProducer) Demand(val int) {
	slf.demand <- val
}

func (slf *MetadataUpdateConsumerProducer) Produce(ctx context.Context, userID string) chan []*queuer.Payload {
	resultsCh := make(chan []*queuer.Payload, 1)

	go func() {
		defer close(resultsCh)

		for {
			select {

			case d := <-slf.demand:
				var results []*queuer.Payload

				for i := 0; i < d; i++ {
					payload, err := slf.queue.ReadMsg(ctx, userID, "metadatasync_consumer")
					if err != nil {
						log.Error().Err(err).Msg("failed to fetch from redis. ignoring")
						return
					}

					if payload != nil {
						results = append(results, payload)
					}
				}

				resultsCh <- results

			case <-ctx.Done():
				return
			}
		}
	}()

	return resultsCh
}

// Create a supervisor like the SSEBroker
// And track each stream for a user
