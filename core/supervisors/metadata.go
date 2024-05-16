package supervisors

import (
	"arbokcore/core/files"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/workerpool"
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type MetadataChangelog struct {
	queue  queuer.Queuer
	demand chan int
}

func NewMetadataChangelog(queue queuer.Queuer) *MetadataChangelog {
	return &MetadataChangelog{
		queue:  queue,
		demand: make(chan int, 1),
	}
}

func (slf *MetadataChangelog) Demand(val int) {
	slf.demand <- val
}

func (slf *MetadataChangelog) Produce(ctx context.Context) chan []*queuer.Payload {
	resultsCh := make(chan []*queuer.Payload, 1)

	go func() {
		defer close(resultsCh)

		for {
			select {

			case d := <-slf.demand:
				var results []*queuer.Payload

				for i := 0; i < d; i++ {
					payload, err := slf.queue.ReadMsg(ctx, "key")
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

type MetadataExecutor struct {
	repo *files.MetadataRepository
}

func NewMetadataExecutor(repo *files.MetadataRepository) *MetadataExecutor {
	return &MetadataExecutor{repo: repo}
}

func (slf *MetadataExecutor) Execute(ctx context.Context, payloads []*queuer.Payload) error {
	//Ideally there should be only 1 here

	fmt.Printf("len of payloads %d, payloads %+v\n", len(payloads), payloads)

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		b := bytes.NewBuffer(payload.Message)

		cachedData := files.CacheMetadata{}

		decoder := gob.NewDecoder(b)
		err := decoder.Decode(&cachedData)
		if err != nil {
			log.Error().Err(err).Msg("failed to decode")
			return err
		}

		err = slf.repo.Update(ctx, cachedData.PrevID, cachedData.ID)
		if err != nil {
			log.Error().Err(err).Msg("failed to update file metadata")
			return err
		}
	}

	log.Info().Msg("metadata file update success")
	return nil
}

func MetadataSupervisor(
	ctx context.Context,
	repo *files.MetadataRepository,
	producer *MetadataChangelog,

) {

	log.Info().Msg("starting metadata changelog supervisor")

	executor := &MetadataExecutor{repo: repo}
	pool := workerpool.NewWorkerPool(2, executor.Execute)

	recvChan := producer.Produce(ctx)
	go workerpool.Dispatch(ctx, pool, recvChan)

	pool.Start(ctx)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		var ticker = time.NewTicker(2 * time.Second)

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("stopping pool")
				pool.Stop(ctx)
				return
			case <-ticker.C:
				producer.Demand(1)
			}
		}
	}()

	wg.Wait()

}
