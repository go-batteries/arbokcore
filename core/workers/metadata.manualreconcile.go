package workers

import (
	"arbokcore/core/database"
	"arbokcore/core/files"
	"arbokcore/core/supervisors"
	"arbokcore/pkg/config"
	"arbokcore/pkg/squirtle"
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
)

func ManualChunkMetadataReconciler(ctx context.Context, cfg config.AppConfig) error {
	conn := database.ConnectSqlite(cfg.DbName)
	dbconn := conn.Connect(ctx)

	ctx = context.Background()

	qs := squirtle.LoadAll("./config/querystore.yaml")

	metadataQueryStore, err := qs.HydrateQueryStore("file_metadatas")
	if err != nil {
		return err
	}

	repo := files.NewMetadataRepository(
		dbconn,
		metadataQueryStore,
	)

	toPtr := func(v string) *string {
		return &v
	}

	chunkQueryStore, err := qs.HydrateQueryStore("user_files")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load user files query")
	}

	crepo := files.NewUserFileRespository(dbconn, chunkQueryStore)

	ids := []*string{
		toPtr("01HY38Y6X2GHW56X39N2Y2T8HQ"),
		toPtr("01HY38X7WT2NS3Z130FXX930E4"),
	}

	filesWithChunks, err := repo.SelectFiles(ctx, ids)
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

	if resps[0].ID == *(ids[0]) {
		thisFile = resps[0]
		prevFile = resps[1]
	} else {
		thisFile = resps[1]
		prevFile = resps[0]
	}

	var fillerChunks map[string]*files.FilesWithChunks

	if thisFile.NChunks == prevFile.NChunks {
		fillerChunks = supervisors.RestOfChunks(thisFile, prevFile)
	}

	fmt.Println("validation", !supervisors.Validate(supervisors.ReconstructChunks(thisFile, prevFile), prevFile))

	if len(fillerChunks) == 0 {
		log.Info().Msg("no matching chunks, file completely replaced")

		err = repo.Update(
			ctx,
			ids[0],
			*ids[1],
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
		err = crepo.CreateBatch(ctx, chunks)
	}

	var uploadStatus = files.StatusCompleted

	if err != nil {
		log.Error().Err(err).Msg("failed to repopulate user files")
		uploadStatus = files.StatusFailed
	}

	log.Info().Str("upload_status", uploadStatus).Msg("updating the current flag now")
	err = repo.Update(
		ctx,
		ids[0],
		*ids[1],
		uploadStatus,
	)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to update file metadata")

		return err
	}

	return nil
}
