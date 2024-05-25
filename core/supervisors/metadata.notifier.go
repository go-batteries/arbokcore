package supervisors

import (
	"arbokcore/pkg/queuer"
	"context"

	"github.com/rs/zerolog/log"
)

type MetadataUpdateConsumer struct {
	queue  queuer.Queuer
	demand chan Demand
}

// Since for each users device, we will be demanding the messages on the
// Q, with each tick in the loop in the /subscribe API.
type Demand struct {
	Count  int
	UserID string
}

func NewMetadataNotifier(queue queuer.Queuer) *MetadataUpdateConsumer {
	return &MetadataUpdateConsumer{
		queue:  queue,
		demand: make(chan Demand, 1),
	}
}

func (slf *MetadataUpdateConsumer) Demand(val Demand) {
	slf.demand <- val
}

func (slf *MetadataUpdateConsumer) Produce(ctx context.Context) chan []*queuer.Payload {
	resultsCh := make(chan []*queuer.Payload, 1)

	go func() {
		defer close(resultsCh)

		for {
			select {

			case d := <-slf.demand:
				var results []*queuer.Payload

				for i := 0; i < d.Count; i++ {
					payload, err := slf.queue.ReadMsg(ctx, d.UserID, "")
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
