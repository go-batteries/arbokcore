package files

import (
	"arbokcore/core/tokens"
	"arbokcore/pkg/squirtle"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

type MetadataRepository struct {
	conn    *sqlx.DB
	querier squirtle.QueryMapper
}

const (
	CreateFileMetadataStmt = "CreateFileMetadata"
	GetFilesForUserStmt    = "GetFilesForUser"
	UpdateFileMetadataStmt = "UpdateFileMetadata"

	InsertFileChunk = "InsertFileChunk"
	GetChunksByFile = "GetChunksByFile"
)

func NewMetadataRepository(conn *sqlx.DB, querier squirtle.QueryMapper) *MetadataRepository {
	return &MetadataRepository{
		conn:    conn,
		querier: querier,
	}
}

// TODO: maybe move the request response objects
// to separate file
// Moving all of them into an api/
// would also work.
// There are pros and cons of both
// For pros, using a separate api/ directory
// would reduce any chance of entering a import cycle
// but then it reduces locality, so for file handling
// the api request response would be in a separate place
// than the files/ directory.
// We will see about this later. Its not as important rn
type MetadataRequest struct {
	UserID       string `json:"-"`
	FileName     string `json:"fileName"`
	FileType     string `json:"fileType"`
	FileSize     int64  `json:"fileSize"` // in bytes
	Digest       string `json:"digest"`
	UploadStatus string `json:"-"`
	Chunks       int    `json:"chunks"`
	AccessToken  string `json:"-"`
}

func (mr *MetadataRepository) Create(
	ctx context.Context,
	metadata *FileMetadata,
) (*FileMetadata, error) {

	stmt, ok := mr.querier.GetQuery(CreateFileMetadataStmt)
	if !ok {
		return nil, errors.New("query_retriever_failed:2001:500")
	}

	_, err := mr.conn.NamedExecContext(ctx, stmt, metadata)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert into database")
		return nil, errors.New("token_generation_failed:2003:500")
	}

	return metadata, nil
}

func (mr *MetadataRepository) Update(
	ctx context.Context,
	metadata *FileMetadata,
) error {

	stmt, ok := mr.querier.GetQuery(UpdateFileMetadataStmt)
	if !ok {
		return errors.New("query_retriever_failed:2004:500")
	}

	_, err := mr.conn.NamedExecContext(
		ctx,
		stmt,
		metadata,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to update file metadata info")
	}

	return err
}

const DefaultLimit = 20

func (mr *MetadataRepository) ListByUserID(
	ctx context.Context,
	userID string,
	offset int,
) ([]*FilesWithChunks, bool, error) {

	hasMore := false

	stmtTmpl, ok := mr.querier.GetQuery(GetFilesForUserStmt)
	if !ok {
		return nil, hasMore, errors.New("query_retriever_failed:2001:500")
	}

	stmt := fmt.Sprintf(stmtTmpl, DefaultLimit+1, offset)
	log.Debug().Msg(stmt)

	rows, err := mr.conn.NamedQueryContext(
		ctx,
		stmt,
		map[string]any{
			"user_id": userID,
		},
	)

	if err != nil {
		log.Error().Err(err).Msg("failed to fetch metadata from database")
		return nil, hasMore, errors.New("database_failed:2002:500")
	}

	var results = []*FilesWithChunks{}

	for rows.Next() {
		metadata := &FilesWithChunks{}
		if err := rows.StructScan(metadata); err != nil {
			log.Error().Err(err).Msg("metadata record unmarshal failed")
			return nil, hasMore, errors.New("record_unmarshal_failed:2003:500")
		}

		results = append(results, metadata)
	}

	hasMore = len(results) > DefaultLimit

	return results, hasMore, nil
}

type MetadataTokenRepository struct {
	conn         *sqlx.DB
	metaQuerier  squirtle.QueryMapper
	tokenQuerier squirtle.QueryMapper
}

func NewMetadataTokenRepository(
	conn *sqlx.DB,
	metaQuerier squirtle.QueryMapper,
	tokenQuerier squirtle.QueryMapper,
) *MetadataTokenRepository {

	return &MetadataTokenRepository{
		conn:         conn,
		metaQuerier:  metaQuerier,
		tokenQuerier: tokenQuerier,
	}
}

func (slf *MetadataTokenRepository) CreateMetadataToken(
	ctx context.Context,
	metadata *FileMetadata,
	token *tokens.Token,
) (err error) {

	log.Info().Msg("create file metadata request and stream token")

	metaCreateStmt, ok := slf.metaQuerier.GetQuery(CreateFileMetadataStmt)
	if !ok {
		log.Error().Msg("failed to create metadata stmt")
		return ErrorStmtNotFound
	}

	tokenStmt, ok := slf.tokenQuerier.GetQuery(tokens.CreateTokenStmt)
	if !ok {
		log.Error().Msg("failed to create tokens stmt")
		return ErrorStmtNotFound
	}

	// Create the tokens and file metadatas in a transaction
	tx, err := slf.conn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.NamedExecContext(
		ctx,
		metaCreateStmt,
		metadata,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create file metadata")
		tx.Rollback()
		return err
	}

	_, err = tx.NamedExecContext(
		ctx,
		tokenStmt,
		token,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create tokens for stream")
		tx.Rollback()

		return err
	}

	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("failed to create metadata or tokens")
		return err
	}

	log.Info().Msg("metadata and tokens created successfully")

	return nil
}
