package supervisors

import (
	"arbokcore/core/database"
	"arbokcore/core/files"
	"arbokcore/core/notifiers"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/utils"
	"arbokcore/pkg/workerpool"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
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
					payload, err := slf.queue.ReadMsg(ctx, "", "")
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
	repo     *files.MetadataRepository
	crepo    *files.UserFileRepository
	notifier *notifiers.MetadataUpdateStatus
}

func NewMetadataExecutor(
	repo *files.MetadataRepository,
	crepo *files.UserFileRepository,
	notifier *notifiers.MetadataUpdateStatus,
) *MetadataExecutor {

	return &MetadataExecutor{repo: repo, crepo: crepo, notifier: notifier}
}

func (slf *MetadataExecutor) ExecuteEach(ctx context.Context, cachedData *files.CacheMetadata) error {
	// At this stage, one upload action has completed
	// So, for a newly update file, Some chunks maybe missing
	// So compare the NChunks with Uploaded chunks in user_files

	// If the expected NChunks < len(Uploaded Chunks)
	// Populate the remaining chunk info from prev fileID
	// Updates the upload status to completed or failed accordingly
	// If prevFileID is null, then mark the upload status as failed
	// Change the file metadata id to current_flag

	// Here we get the file information with chunks for the PrevFileID and NewFileID
	// If this is a new file, then there is nothing to compare,
	// set the current_flag to 1 for NewFileID

	// If not, check for the NChunks of the Previous and NewFiles' metadata
	// If they are same, check if we need to reconstruct the file chunks
	// If not, the file is complete new version.
	// set the current_flag to 1 for NewFileID
	// set the current_flag to 0 for PrevFileID and the end_date to present date.

	ids := []*string{&cachedData.ID, cachedData.PrevID}

	// fmt.Println("===cached data====")
	// utils.Dump(cachedData)

	//TODO: Add tests for refactoring
	if cachedData.PrevID == nil {
		err := slf.repo.Update(
			ctx,
			cachedData.PrevID,
			cachedData.ID,
			files.StatusCompleted,
		)
		if err != nil {
			log.Error().
				Err(err).
				Msg("failed to update file metadata")

			return err
		}

		return nil
	}

	filesWithChunks, err := slf.repo.SelectFiles(ctx, ids)
	if err != nil {
		log.Error().Err(err).Msg("failed to get files by id")
		return err
	}

	resps := files.BuildFilesInfoResponse(filesWithChunks)

	log.Info().
		Int("count", len(resps)).
		Msg("total files with chunks")

	if len(resps) < 2 {
		log.Info().Msg("prev chunk not found, so probably some error")
		return errors.New("file_merge_conflict")
	}

	var (
		thisFile *files.FileInfoResponse
		prevFile *files.FileInfoResponse
	)

	// Since there are only two records requested
	// Simple if to determine which file is prev vs which is new
	// Incase data arrives out of order
	if resps[0].ID == *(ids[0]) {
		thisFile = resps[0]
		prevFile = resps[1]
	} else {
		thisFile = resps[1]
		prevFile = resps[0]
	}

	var fillerChunks map[string]*files.FilesWithChunks

	if thisFile.NChunks == prevFile.NChunks {
		fillerChunks = RestOfChunks(thisFile, prevFile)
	}

	isValid := Validate(ReconstructChunks(thisFile, prevFile), prevFile)
	log.Info().Bool("isvalid", isValid).Msg("file reconstruction validation")

	if len(fillerChunks) == 0 {
		log.Info().Msg("no matching chunks, file completely replaced")

		err = slf.repo.Update(
			ctx,
			cachedData.PrevID,
			cachedData.ID,
			files.StatusCompleted,
		)
		if err != nil {
			log.Error().
				Err(err).
				Msg("failed to update file metadata")

			return err
		}

		return nil
	}

	chunks := []*files.UserFile{}

	for _, chunk := range fillerChunks {
		uf := &files.UserFile{
			UserID:       thisFile.UserID,
			FileID:       thisFile.ID,
			ChunkID:      chunk.ChunkID,
			ChunkBlobUrl: chunk.ChunkBlobUrl,
			ChunkHash:    chunk.ChunkHash,
			NextChunkID:  chunk.NextChunkID,
			Timestamp:    database.NewTimestamp(),
		}
		chunks = append(chunks, uf)
	}

	if len(chunks) > 0 {
		log.Info().Int("count", len(chunks)).Msg("need to create older chunks")
		err = slf.crepo.CreateBatch(ctx, chunks)
	}

	var uploadStatus = files.StatusCompleted

	if err != nil {
		log.Error().Err(err).Msg("failed to repopulate user files")
		uploadStatus = files.StatusFailed
	}

	log.Info().Str("upload_status", uploadStatus).Msg("updating the current flag now")
	err = slf.repo.Update(
		ctx,
		cachedData.PrevID,
		cachedData.ID,
		uploadStatus,
	)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to update file metadata")
	}

	return err
}

func (slf *MetadataExecutor) Execute(ctx context.Context, payloads []*queuer.Payload) error {
	//Ideally there should be only 1 here
	fmt.Printf("len of payloads %d, payloads %+v\n", len(payloads), payloads)
	defer utils.Bench2("metadata_executor")()

	updateSuccessEvents := []*notifiers.MetadataUpdateStatusEvent{}

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

		err = slf.ExecuteEach(ctx, &cachedData)
		if err != nil {
			return err
		}

		fmt.Println("sending to notify")
		utils.Dump(cachedData)

		updateSuccessEvents = append(updateSuccessEvents, &notifiers.MetadataUpdateStatusEvent{
			FileID:   cachedData.ID,
			UserID:   cachedData.UserID,
			DeviceID: cachedData.DeviceID,
		})
	}

	// From here on, once the sync is complete
	// We need a notifier, so that the file sync on client side
	// can complete
	// This involves, getting the notification on the update status
	// for a given userID and fileID.

	// We can again use a message queue, (I am going to use the existing redis queue)
	// The notification workers responsibility is
	// 1. Figure out which users shall receive the file updates.
	// 2. Call the API server internal endpoint (Because the gateway is using consistent hashing)
	//    It should automatically route to the appropriate server for the userID

	err := slf.notifier.Notify(ctx, updateSuccessEvents)
	if err != nil {
		fmt.Println("failed to notify with err", err)
	}
	return err
}

func RestOfChunks(thisFile, prevFile *files.FileInfoResponse) map[string]*files.FilesWithChunks {
	matchedChunks := map[string]*files.FilesWithChunks{}

	log.Info().Msg("checking for matching chunks")

	// If currentChunkID is missing in the lastValiFileChunks
	// Add that to return values
	for chunkIDstr, chunk := range prevFile.Chunks {
		_, ok := thisFile.Chunks[chunkIDstr]
		if ok {
			continue
		}

		//FIX: have some more checks to validate the sequence
		// So maybe first recreate the file chunks
		// And validate the chunk sequence.
		// Record with NextChunkID == -1 should not change

		matchedChunks[chunkIDstr] = chunk
	}

	// fmt.Println("matching chunks")
	// utils.Dump(matchedChunks)

	return matchedChunks
}

// Validate the reconstructed file against the previous file
func ReconstructChunks(thisFile, prevFile *files.FileInfoResponse) *files.FileInfoResponse {
	missingChunks := RestOfChunks(thisFile, prevFile)

	for id, chunk := range missingChunks {
		thisFile.Chunks[id] = chunk
	}

	return thisFile
}

func Validate(reconstructedChunk, prevChunk *files.FileInfoResponse) bool {
	if prevChunk.NChunks != reconstructedChunk.NChunks {
		return false
	}

	prevChunkChain := map[string]string{}

	for id, chunk := range prevChunk.Chunks {
		if chunk.NextChunkID == nil {
			return false
		}

		prevChunkChain[id] = fmt.Sprintf("%d", *chunk.NextChunkID)
	}

	for id, chunk := range reconstructedChunk.Chunks {
		if chunk.NextChunkID == nil {
			return false
		}

		if prevChunkChain[id] != fmt.Sprintf("%d", *chunk.NextChunkID) {
			return false
		}
	}

	return true
}

// func ValidateFileChunksChain(fileChunks *files.FileInfoResponse) bool {}

func MetadataSupervisor(
	ctx context.Context,
	repo *files.MetadataRepository,
	crepo *files.UserFileRepository,
	producer *MetadataChangelog,
	notifier *notifiers.MetadataUpdateStatus,

) {

	log.Info().Msg("starting metadata changelog supervisor")

	// This is the processorFunc for each worker
	executor := NewMetadataExecutor(repo, crepo, notifier)

	// WorkerPool that will Process each Redis Message
	pool := workerpool.NewWorkerPool(10, executor.Execute)
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
				log.Info().Msg("demand 1")
				producer.Demand(1)
			}
		}
	}()

	wg.Wait()

}
