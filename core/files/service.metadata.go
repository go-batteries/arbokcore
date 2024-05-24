package files

import (
	"arbokcore/core/api"
	"arbokcore/core/database"
	"arbokcore/core/tokens"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/rho"
	"arbokcore/pkg/utils"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type MetadataService struct {
	repo           *MetadataRepository
	metaTokensRepo *MetadataTokenRepository
	queue          queuer.Queuer
}

func NewMetadataService(
	repo *MetadataRepository,
	metaTokensRepo *MetadataTokenRepository,
	queue queuer.Queuer,
) *MetadataService {

	return &MetadataService{
		repo:           repo,
		metaTokensRepo: metaTokensRepo,
		queue:          queue,
	}
}

func (ms MetadataService) ListOrderedFileChunks(ctx context.Context, fileID string, userID string) ([]string, *FileInfoResponse, error) {

	filesWithChunks, err := ms.repo.SelectFiles(ctx, []*string{&fileID})
	if err != nil {
		log.Error().Err(err).Msg("failed to get files with chunks")
		return nil, nil, errors.New("internal_error:2025:500")
	}

	filesWithChunks = rho.Filter(filesWithChunks, func(chunk *FilesWithChunks, _ int) bool {
		return chunk.UserID == userID
	})

	metadataResponses := BuildFilesInfoResponse(filesWithChunks)
	if len(metadataResponses) < 1 {
		return nil, nil, errors.New("file_corrupted:2045:500")
	}

	var fileInfoResp *FileInfoResponse
	for _, resp := range metadataResponses {
		if resp.ID == fileID {
			fileInfoResp = resp
		}
	}

	if fileInfoResp == nil {
		return nil, nil, errors.New("file_not_found:2046:500")
	}

	chunks := make([]string, fileInfoResp.NChunks)

	for _, chunk := range fileInfoResp.Chunks {
		chunks[chunk.ChunkID] = chunk.ChunkBlobUrl
	}

	log.Info().Int("chunks len", len(chunks)).Msg("number of chunks")

	return chunks, fileInfoResp, nil
}

func (ms *MetadataService) MarkUploadComplete(
	ctx context.Context,
	fileID string,
) api.Response {

	log.Info().Msg("marking file upload as complete")

	results, err := ms.repo.FindBy(ctx, FindClause{
		{Key: "id", Operator: "=", Val: fileID},
		{Key: "upload_status", Operator: "=", Val: StatusUploading},
	})
	if err != nil || len(results) == 0 {
		return api.BuildResponse(errors.New("file_not_found:2019:404"), nil)
	}

	metadata := results[0]
	// utils.Dump(metadata)

	// Enqueue the PreviousFile ID and File.ID as Message

	// Since this request is sent at the end of the chunk upload request
	// The assumption is that, the fileID is already present
	// In the file_metadatas table.

	// So while creating the file_metadatas record for
	// New version of the file, The new FileID is created
	// With the PreviousFileID record and populated in DB
	qdata := CacheMetadata{
		UserID: metadata.UserID,
		PrevID: metadata.PrevID,
		ID:     fileID,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err = enc.Encode(qdata)
	if err == nil {
		err = ms.queue.EnqueueMsg(ctx, "", &queuer.Payload{
			Message: buf.Bytes(),
		})
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to encode or enqueue message")
		return api.BuildResponse(errors.New("internal_error:2021:500"), nil)
	}

	return api.BuildResponse(nil, map[string]any{"eof": true})

}

func (ms *MetadataService) PrepareFileForUpload(
	ctx context.Context,
	req MetadataRequest,
) api.Response {

	log.Info().Msg("adding file metadata to db")

	chunks := CalculateChunks(req.FileSize)

	found, err := ms.repo.FindByHash(ctx, req.Digest)
	if err != nil {
		log.Error().Err(err).Msg("failed to execute query")

		if found {
			err = ErrDuplicateFile
		}
	}
	if err != nil {
		return api.BuildResponse(errors.New("duplicate:2014:422"), nil)
	}

	id, err := database.NewID()
	if err != nil {
		log.Error().Err(err).Msg("faild to generate ulid")
		return api.BuildResponse(errors.New("id_gen_failed:2002:500"), nil)
	}

	if chunks != req.Chunks {
		log.Error().Msg("number of chunk mismatch")
		return api.BuildResponse(errors.New("chunks_size_invalid:2005:422"), nil)
	}

	metadata := &FileMetadata{
		ID:          id,
		UserID:      req.UserID,
		Filename:    req.FileName,
		FileSize:    req.FileSize,
		FileType:    req.FileType,
		FileHash:    req.Digest,
		NChunks:     int(chunks),
		UploadStaus: req.UploadStatus,
		CurrentFlag: false,
		Timestamp:   database.NewTimestamp(),
	}

	token, err := tokens.NewToken(
		metadata.ID, tokens.ResourceTypeStream,
		tokens.WithDefaultExpiry(),
		tokens.WithUserID(&req.UserID),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize token")
		return api.BuildResponse(
			errors.New("internal_server_error:2010:500"),
			nil,
		)
	}

	// We build the Token and FileMetadata records to put in DB
	if err := ms.metaTokensRepo.CreateMetadataToken(
		ctx,
		metadata,
		token,
	); err != nil {
		log.Error().Err(err).Msg("failed to create metadata and token")
		return api.BuildResponse(
			errors.New("internal_server_error:2011:500"),
			nil,
		)
	}

	return api.BuildResponse(nil, &MetadataTokenResponse{
		StreamToken:  token.AccessToken,
		FileID:       id,
		UploadStatus: StatusUploading,
		CreatedAt:    token.CreatedAt,
		ExpiresIn:    tokens.ShortExpiryDuration,
		Digest:       metadata.FileHash,
		FileType:     metadata.FileType,
	})
}

type MetadataTokenResponse struct {
	AccessToken  string        `json:"accessToken,omitempty"`
	StreamToken  string        `json:"streamToken,omitempty"`
	PrevID       *string       `json:"prevID"`
	FileID       string        `json:"fileID"`
	FileType     string        `json:"fileType"`
	Digest       string        `json:"fileHash"`
	UploadStatus string        `json:"uploadStatus"`
	CreatedAt    time.Time     `json:"createdAt"`
	ExpiresIn    time.Duration `json:"expiresAt"`
}

// Get the previous fileID from :fileID
// Create a new record with new fileID
// Return the new fileID in response
func (ms *MetadataService) UpdateFileMetadata(
	ctx context.Context,
	req *api.FileUpdateMetadataRequest,
) api.Response {
	// So here the update also needs to
	// Create a new token and return the same response as Create
	chunks := CalculateChunks(req.FileSize)
	if chunks != int(req.Chunks) {
		return api.BuildResponse(
			errors.New("invalid_file_data:2006:422"),
			nil,
		)
	}

	results, err := ms.repo.FindBy(ctx, FindClause{
		{Key: "id", Operator: "=", Val: req.FileID},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to find file by id")
		return api.BuildResponse(errors.New("file_not_found:2022:422"), nil)
	}

	if len(results) == 0 {
		log.Info().Msg("not files found")
		return api.BuildResponse(errors.New("file_not_found:2023:404"), nil)
	}

	prevMetadata := results[0]

	id, err := database.NewID()
	if err != nil {
		log.Error().Err(err).Msg("faild to generate ulid")
		return api.BuildResponse(errors.New("id_gen_failed:2002:500"), nil)
	}

	log.Info().Msg("updating file metadata")
	utils.Dump(req)

	if prevMetadata.FileHash == req.Digest {
		return api.BuildResponse(errors.New("duplicate:2014:422"), nil)
	}

	log.Info().Str("prevm", prevMetadata.ID).Str("reqfi", req.FileID)

	metadata := &FileMetadata{
		ID:          id,
		PrevID:      &req.FileID,
		UserID:      req.UserID,
		FileSize:    req.FileSize,
		FileType:    prevMetadata.FileType,
		Filename:    prevMetadata.Filename,
		FileHash:    req.Digest,
		NChunks:     int(req.Chunks),
		CurrentFlag: false,
		UploadStaus: StatusUploading,
		Timestamp:   database.NewTimestamp(),
	}

	// fmt.Printf("updated metadata %+v\n", metadata)

	token, err := tokens.NewToken(
		metadata.ID, tokens.ResourceTypeStream,
		tokens.WithDefaultExpiry(),
		tokens.WithUserID(&req.UserID),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize token")
		return api.BuildResponse(
			errors.New("internal_server_error:2011:500"),
			nil,
		)
	}

	if err := ms.metaTokensRepo.CreateMetadataToken(
		ctx,
		metadata,
		token,
	); err != nil {
		log.Error().Err(err).Msg("failed to create metadata and token")
		return api.BuildResponse(
			errors.New("internal_server_error:2011:500"),
			nil,
		)
	}

	return api.BuildResponse(nil, &MetadataTokenResponse{
		StreamToken:  token.AccessToken,
		PrevID:       metadata.PrevID,
		FileID:       metadata.ID,
		CreatedAt:    token.CreatedAt,
		UploadStatus: StatusUploading,
		ExpiresIn:    tokens.ShortExpiryDuration,
	})
}

type FileInfoResponse struct {
	ID          string `json:"fileID"`
	Name        string `json:"fileName"`
	Hash        string `json:"fileHash"`
	Size        int64  `json:"fileSize"`
	Type        string `json:"fileType"`
	CurrentFlag bool   `json:"currentFlag"`

	Chunks map[string]*FilesWithChunks `json:"chunks"`

	NChunks int    `json:"nChunks"`
	UserID  string `json:"userID"`
}

type Response struct {
	FilesInfo []*FileInfoResponse `json:"files"`
	HasMore   bool                `json:"hasMore"`
}

func (ms *MetadataService) ListFilesForUser(
	ctx context.Context,
	userID string,
	offset int,
) api.Response {
	// This gets the files and their chunks info from file_metadatas and user_files
	// So the response comes like
	// FileInfo1 | Chunk1
	// FileInfo1 | Chunk2
	// FileInfo3 | Chunk1
	// FileInfo3 | Chunk2

	// We need to convert this to
	// File: { FileInfo, Chunks: [Chunk1, Chunk2]}
	files, hasMore, err := ms.repo.ListByUserID(ctx, userID, offset)
	if err != nil {
		return api.BuildResponse(err, nil)
	}

	if len(files) == 0 {
		log.Info().Msg("no files for user")

		return api.BuildResponse(nil, Response{})
	}

	filesInfo := BuildFilesInfoResponse(files)

	return api.BuildResponse(nil, Response{FilesInfo: filesInfo, HasMore: hasMore})
}

func BuildFilesInfoResponse(files []*FilesWithChunks) []*FileInfoResponse {
	filesInfo := []*FileInfoResponse{}
	filesSeenIndex := map[string]int{}
	i := 0

	for _, file := range files {
		fileInfo := &FileInfoResponse{
			ID:          file.ID,
			Name:        file.Filename,
			Hash:        file.FileHash,
			Size:        file.FileSize,
			Type:        file.FileType,
			CurrentFlag: file.CurrentFlag,
			NChunks:     file.NChunks,
			UserID:      file.UserID,

			Chunks: make(map[string]*FilesWithChunks),
		}

		// We are using string here, so that going forward
		// If we want we can change the chunk_id to be a generated id
		// instead of number.
		versionStr := fmt.Sprintf("%d", file.ChunkID)

		arrayIDx, ok := filesSeenIndex[file.ID]
		if !ok {
			filesSeenIndex[file.ID] = i
			fileInfo.Chunks[versionStr] = file
			filesInfo = append(filesInfo, fileInfo)

			i += 1
		} else {
			f := filesInfo[arrayIDx]
			f.Chunks[versionStr] = file
		}
	}

	return filesInfo
}
