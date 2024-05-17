package files

import (
	"arbokcore/core/api"
	"arbokcore/core/database"
	"arbokcore/pkg/blobstore"
	"arbokcore/pkg/squirtle"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

const (
	CreateFileChunkStmt = "CreateFileChunk"
	GetFileChunksStmt   = "GetFileChunks"
	GetFilesChunksStmt  = "GetFilesChunks"
	GetChunkForFileStmt = "GetChunkForFile"
)

type UserFileRepository struct {
	conn    *sqlx.DB
	querier squirtle.QueryMapper
}

func NewUserFileRespository(conn *sqlx.DB, querier squirtle.QueryMapper) *UserFileRepository {
	return &UserFileRepository{
		conn:    conn,
		querier: querier,
	}
}

var (
	ErrorStmtNotFound = errors.New("stmt_not_found")
)

// TODO: Put this chunk in redis store (redis-sentinel)
// And then use a queue (LPUSH/BLPOP) or kafka to signal to statrt processing file
// More than a redis, we need a durable append only log file like thing
// Because copying this chunks to s3 will take time and we don't want them blocked
// For now, we will just store in s3/local fs and then write the url to database.

func (slf *UserFileRepository) Create(ctx context.Context, userFile *UserFile) error {
	stmt, ok := slf.querier.GetQuery(CreateFileChunkStmt)
	if !ok {
		return ErrorStmtNotFound
	}

	//TODO: validate userFile

	_, err := slf.conn.NamedExecContext(ctx, stmt, userFile)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert into database")
	}

	return err
}

func (slf *UserFileRepository) CreateBatch(ctx context.Context, chunks []*UserFile) error {
	stmt, ok := slf.querier.GetQuery(CreateFileChunkStmt)
	if !ok {
		return ErrorStmtNotFound
	}

	//TODO: validate userFile

	tx, err := slf.conn.BeginTxx(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to init transaction")
		return err
	}

	for _, chunk := range chunks {
		_, err = tx.NamedExecContext(ctx, stmt, chunk)
		if err != nil {
			log.Error().Err(err).Msg("failed to prepare transaction")
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

type FileChunkRequest struct {
	UserID      string        `json:"-"`
	FileID      string        `json:"-"`
	ChunkID     int64         `json:"chunkID"`
	ChunkData   io.ReadCloser `json:"-"`
	NextChunkID *int64        `json:"nextChunkID,omitempty"`

	// We will come back to this later
	Version string `json:"-"`
}

func (slf *UserFileRepository) GetChunksForFile(ctx context.Context, fileID string) error {
	stmt, ok := slf.querier.GetQuery(GetFileChunksStmt)
	if !ok {
		return errors.New("query_retriever_failed:4001:500")
	}

	// Validate chunk

	dest := []*UserFile{}

	err := slf.conn.SelectContext(ctx, dest, stmt, map[string]string{"file_id": fileID})
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch chunks from db")
		return errors.New("internal_error:4002:500")
	}

	return err
}

func (slf *UserFileRepository) GetChunk(ctx context.Context, userFile *UserFile) error {
	stmt, ok := slf.querier.GetQuery(GetChunkForFileStmt)
	if !ok {
		return errors.New("query_retriever_failed:4001:500")
	}

	// Validate chunk

	dest := &UserFile{}

	err := slf.conn.GetContext(ctx, dest, stmt, userFile)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch chunks from db")
		return errors.New("internal_error:4002:500")
	}

	return err
}

type FileChunkService struct {
	repo    *UserFileRepository
	storage blobstore.BlobStorage
}

func NewFileChunkService(repo *UserFileRepository, blobstore blobstore.BlobStorage) *FileChunkService {
	return &FileChunkService{repo: repo, storage: blobstore}
}

func ValidateHash(ctx context.Context, req *api.FileChunkRequest) error {
	hash := sha256.New()

	_, err := io.Copy(hash, req.Data)
	if err != nil {
		return err
	}

	result := []byte{}
	digest := hex.EncodeToString(hash.Sum(result))

	if req.ChunkDigest != digest {
		return errors.New("hash_mismatch")
	}

	return nil
}

func (slf *FileChunkService) Create(ctx context.Context, req *api.FileChunkRequest) api.Response {
	if err := ValidateHash(ctx, req); err != nil {
		log.Error().Err(err).Msg("hash mismatch in chunk")

		return api.BuildResponse(errors.New("corrupted_file:5001:422"), nil)
	}

	chunkIDInt, err := strconv.Atoi(req.ChunkID)
	if err != nil {
		log.Error().Err(err).Msg("chunk id should be integer")

		return api.BuildResponse(errors.New("invalid_file_chunk:5002:422"), nil)
	}

	nextChunkIDInt, err := strconv.ParseInt(req.NextChunkID, 10, 64)
	if err != nil {
		return api.BuildResponse(errors.New("invalid_file_chunk:5004:422"), nil)
	}

	storedPath, err := slf.storage.UpdateChunk(
		ctx,
		req.FileID,
		blobstore.NewChunkedFile(req.Data, int64(chunkIDInt), nil))

	if err != nil {
		log.Error().Err(err).Msg("failed to save chunk to disk")
		return api.BuildResponse(errors.New("internal_server_error:5003:500"), nil)
	}

	model := &UserFile{
		UserID:       req.UserID,
		FileID:       req.FileID,
		ChunkID:      int64(chunkIDInt),
		ChunkBlobUrl: storedPath,
		ChunkHash:    req.ChunkDigest,
		NextChunkID:  &nextChunkIDInt,
		Timestamp:    database.NewTimestamp(),
	}

	if err := slf.repo.Create(ctx, model); err != nil {
		log.Error().Err(err).Msg("failed to save file chunk info to db")
		return api.BuildResponse(errors.New("save_chunk_failed:5005:500"), nil)
	}

	return api.BuildResponse(nil, ChunkUploadResponse{
		ChunkID:      model.ChunkID,
		NextChunkID:  nextChunkIDInt,
		ChunkBlobUrl: storedPath,
		ChunkHash:    req.ChunkDigest,
		CreatedAt:    model.Timestamp.CreatedAt,
		UpdatedAt:    model.Timestamp.UpdatedAt,
	})
}

type ChunkUploadResponse struct {
	ChunkID      int64     `json:"chunkID"`
	NextChunkID  int64     `json:"nextChunkID"`
	ChunkBlobUrl string    `json:"chunkBlobUrl"`
	ChunkHash    string    `json:"chunkHash"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
