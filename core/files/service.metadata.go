package files

import (
	"arbokcore/core/api"
	"arbokcore/core/database"
	"arbokcore/core/tokens"
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

type MetadataService struct {
	repo           *MetadataRepository
	metaTokensRepo *MetadataTokenRepository
}

func NewMetadataService(
	repo *MetadataRepository,
	metaTokensRepo *MetadataTokenRepository,
) *MetadataService {

	return &MetadataService{
		repo:           repo,
		metaTokensRepo: metaTokensRepo,
	}
}

func (ms *MetadataService) PrepareFileForUpload(
	ctx context.Context,
	req MetadataRequest,
) api.Response {

	log.Info().Msg("adding file metadata to db")

	chunks := CalculateChunks(req.FileSize)

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
		StreamToken: token.AccessToken,
		FileID:      id,
		CreatedAt:   token.CreatedAt,
		ExpiresIn:   tokens.ShortExpiryDuration,
	})
}

type MetadataTokenResponse struct {
	AccessToken string        `json:"accessToken,omitempty"`
	StreamToken string        `json:"streamToken,omitempty"`
	FileID      string        `json:"fileID"`
	CreatedAt   time.Time     `json:"createdAt"`
	ExpiresIn   time.Duration `json:"expiresAt"`
}

func (ms *MetadataService) UpdateFileMetadata(
	ctx context.Context,
	req *api.FileUpdateMetadataRequest,
) api.Response {

	chunks := CalculateChunks(req.FileSize)
	if chunks != int(req.Chunks) {
		return api.BuildResponse(
			errors.New("invalid_file_data:2006:422"),
			nil,
		)
	}

	log.Info().Msg("updating file metadata")
	metadata := &FileMetadata{
		ID:          req.FileID,
		FileSize:    req.FileSize,
		FileHash:    req.Digest,
		NChunks:     int(req.Chunks),
		UploadStaus: StatusUploading,
		Timestamp: database.Timestamp{
			UpdatedAt: database.Now(),
		},
	}

	err := ms.repo.Update(ctx, metadata)
	if err != nil {
		log.Error().Err(err).Msg("failed to update file metadata")
		return api.BuildResponse(err, nil)
	}

	return api.BuildResponse(nil, map[string]any{"id": req.FileID, "upload": true})
}

type FileInfoResponse struct {
	ID   string `json:"fileID"`
	Name string `json:"fileName"`
	Hash string `json:"fileHash"`
	Size int64  `json:"fileSize"`
	Type string `json:"fileType"`

	Chunks []*FilesWithChunks `json:"chunks"`
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
			ID:   file.ID,
			Name: file.Filename,
			Hash: file.FileHash,
			Size: file.FileSize,
			Type: file.FileType,

			Chunks: []*FilesWithChunks{file},
		}

		arrayIDx, ok := filesSeenIndex[file.ID]
		if !ok {
			filesSeenIndex[file.ID] = i
			filesInfo = append(filesInfo, fileInfo)
			i += 1
		} else {
			f := filesInfo[arrayIDx]
			f.Chunks = append(f.Chunks, file)
		}
	}

	return filesInfo
}
