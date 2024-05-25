package notifiers

import (
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/rho"
	"bytes"
	"context"
	"encoding/gob"

	"github.com/rs/zerolog/log"
)

type MetadataUpdateStatus struct {
	queue queuer.Queuer
}

// New File Updated Notification queue
func NewMedataUpdateStatus(queue queuer.Queuer) *MetadataUpdateStatus {
	return &MetadataUpdateStatus{
		queue: queue,
	}
}

type MetadataUpdateStatusEvent struct {
	FileID     string
	UserID     string
	PrevFileID *string
	DeviceID   string
}

type PayloadMap struct {
	payload queuer.Payload
	userID  string
}

func (ms *MetadataUpdateStatus) Notify(ctx context.Context, events []*MetadataUpdateStatusEvent) error {
	payloads := rho.Map(
		events,
		func(data *MetadataUpdateStatusEvent, _ int) *PayloadMap {
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)

			err := enc.Encode(data)
			if err != nil {
				log.Error().Err(err).Msg("failed to encode event")
				return nil
			}

			return &PayloadMap{
				payload: queuer.Payload{Message: buf.Bytes()},
				userID:  data.UserID,
			}
		})

	log.Info().Int("events_count", len(events)).
		Int("payloads_count", len(payloads)).
		Msg("total payloads from events")

	log.Info().Int("payloads_count", len(payloads)).Msg("payload count after filter")

	perr := queuer.PartialError{}

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		// Enqueue fileID changes for user device notifications
		err := ms.queue.EnqueueMsg(ctx, payload.userID, &payload.payload)
		if err != nil {
			log.Error().Err(err).Msg("failed to enqueue messages")
			perr.Failed = append(perr.Failed, &payload.payload)
		} else {
			log.Info().Msg("successfully enqueued messages")
		}
	}

	if len(perr.Failed) == 0 {
		return nil
	}

	return perr
}
