package files

import (
	"arbokcore/core/database"
	"arbokcore/core/tokens"
	"arbokcore/pkg/squirtle"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	UpdateCurrentFlagStmt  = "UpdateCurrentFlag"
	FindByHashStmt         = "FindByHash"
	SelectFilesForUserStmt = "SelectFilesForUser"

	InsertFileChunk = "InsertFileChunk"
	GetChunksByFile = "GetChunksByFile"
)

func NewMetadataRepository(conn *sqlx.DB, querier squirtle.QueryMapper) *MetadataRepository {
	return &MetadataRepository{
		conn:    conn,
		querier: querier,
	}
}

type FileIDsRequest struct {
	FileIDs []string `json:"fileIDs"`
}

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

func (slf *MetadataRepository) Create(
	ctx context.Context,
	metadata *FileMetadata,
) (*FileMetadata, error) {

	stmt, ok := slf.querier.GetQuery(CreateFileMetadataStmt)
	if !ok {
		return nil, errors.New("query_retriever_failed:2001:500")
	}

	_, err := slf.conn.NamedExecContext(ctx, stmt, metadata)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert into database")
		return nil, errors.New("token_generation_failed:2003:500")
	}

	return metadata, nil
}

type FindKV struct {
	Key         string
	PlaceHolder string
	Operator    string
	Val         any
}

type OrderKV struct {
	Field string
	Order string
}

type FindClause []FindKV
type OrderClause []OrderKV

type LimitClause struct {
	Limit  int
	Offset int
}

//TODO: Do this instead for FindBy

type QueryBuilder struct {
	*FindClause
	*OrderClause
	*LimitClause
}

func (clauses FindClause) String() string {
	c := []string{}

	for _, clause := range clauses {
		placeholder := clause.PlaceHolder
		if placeholder == "" {
			placeholder = fmt.Sprintf(":%s", clause.Key)
		}

		c = append(c, fmt.Sprintf("%s %s %s", clause.Key, clause.Operator, placeholder))
	}

	return strings.Join(c, " AND ")
}

func (clauses FindClause) Args() map[string]any {
	args := map[string]any{}

	for _, kv := range clauses {
		args[kv.Key] = kv.Val
	}

	return args
}

func (slf *MetadataRepository) FindBy(ctx context.Context, clauses FindClause) ([]*FileMetadata, error) {
	if len(clauses) == 0 {
		return nil, errors.New("clause is not present")
	}

	clauseStr := clauses.String()
	log.Info().Str("clause", clauseStr).Msg("clause str")

	stmtTmpl, ok := slf.querier.GetQuery("FindBy")
	if !ok {
		log.Error().Msg("failed to find by " + clauseStr)
		return nil, ErrorStmtNotFound
	}

	metadatas := []*FileMetadata{}

	stmt := fmt.Sprintf(stmtTmpl, clauseStr)

	fmt.Println("find by ", stmt)

	nstmt, err := slf.conn.PrepareNamedContext(ctx, stmt)
	if err != nil {
		log.Error().Err(err).Msg("failed to prepare named stmt")
		return nil, errors.New("failed to construct prepared stmt")
	}

	err = nstmt.SelectContext(
		ctx,
		&metadatas,
		clauses.Args(),
	)

	if err != nil {
		log.Error().Err(err).Msg("failed to find metadata")
		return nil, err
	}

	log.Info().Int("count", len(metadatas)).Msg("fetched metadatas")
	return metadatas, nil
}

var (
	ErrDuplicateFile = errors.New("duplicate_file")
)

func (slf *MetadataRepository) FindByHash(
	ctx context.Context,
	hashStr string,
) (bool, error) {

	log.Info().Msg("finding by hash")

	stmt, ok := slf.querier.GetQuery(FindByHashStmt)
	if !ok {
		return false, ErrorStmtNotFound
	}

	var found int64

	err := slf.conn.GetContext(
		ctx,
		&found,
		stmt,
		hashStr,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to find by hash")
		return false, nil
	}
	if found > 0 {
		err = ErrDuplicateFile
	}

	return found > 0, err
}

func (mr *MetadataRepository) SelectFiles(ctx context.Context, ids []*string) ([]*FilesWithChunks, error) {

	stmt, ok := mr.querier.GetQuery(SelectFilesForUserStmt)
	if !ok {
		return nil, ErrorStmtNotFound
	}

	query, args, err := sqlx.In(stmt, ids)
	if err != nil {
		log.Error().Err(err).Msg("failed to build in query")
		return nil, err
	}

	query = mr.conn.Rebind(query)
	fmt.Println(query, args)

	filesWithChunks := []*FilesWithChunks{}

	err = mr.conn.SelectContext(
		ctx,
		&filesWithChunks,
		query,
		args...,
	)

	log.Info().Int("count", len(filesWithChunks)).Msg("fetched files with chunks")

	return filesWithChunks, err
}

func (mr *MetadataRepository) Update(
	ctx context.Context,
	prevFileID *string,
	newFileID string,
	uploadStatus string,
) (err error) {

	log.Info().Msg("mark record as current")

	tx, err := mr.conn.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})

	stmtTmpl, ok := mr.querier.GetQuery(UpdateCurrentFlagStmt)
	if !ok {
		return ErrorStmtNotFound
	}

	newRecord := map[string]any{
		"current_flag":  1,
		"end_date":      nil,
		"id":            newFileID,
		"prev_id":       prevFileID,
		"upload_status": uploadStatus,
	}

	stmt := fmt.Sprintf(stmtTmpl, "AND prev_id IS :prev_id")
	log.Debug().Str("update", "new").Any("rec", newRecord).Msg(stmt)

	_, err = tx.NamedExecContext(
		ctx,
		stmt,
		newRecord,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	if prevFileID == nil {
		return tx.Commit()
	}

	oldRecord := map[string]any{
		"current_flag":  0,
		"end_date":      database.Now(),
		"id":            *prevFileID,
		"upload_status": uploadStatus,
	}

	stmt = fmt.Sprintf(stmtTmpl, "")
	log.Debug().Str("update", "old").Any("rec", newRecord).Msg(stmt)

	_, err = tx.NamedExecContext(
		ctx,
		stmt,
		oldRecord,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

const DefaultLimit = 20

func (slf *MetadataRepository) ListByUserID(
	ctx context.Context,
	userID string,
	offset int,
) ([]*FilesWithChunks, bool, error) {

	hasMore := false

	stmtTmpl, ok := slf.querier.GetQuery(GetFilesForUserStmt)
	if !ok {
		return nil, hasMore, errors.New("query_retriever_failed:2001:500")
	}

	stmt := fmt.Sprintf(stmtTmpl, DefaultLimit+1, offset)
	log.Debug().Msg(stmt)

	rows, err := slf.conn.NamedQueryContext(
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

// Like Create, It updates the metadata info and creates a new stream token
func (slf *MetadataTokenRepository) UpdateMetadataToken(
	ctx context.Context,
	metadata *FileMetadata,
	token *tokens.Token,
) (err error) {

	log.Info().Msg("update file metadata request and create stream token")

	// metaCreateStmt, ok := slf.metaQuerier.GetQuery(UpdateFileMetadataStmt)
	// if !ok {
	// 	log.Error().Msg("failed to create metadata update stmt")
	// 	return ErrorStmtNotFound
	// }
	//
	// tokenStmt, ok := slf.tokenQuerier.GetQuery(tokens.CreateTokenStmt)
	// if !ok {
	// 	log.Error().Msg("failed to create tokens stmt")
	// 	return ErrorStmtNotFound
	// }
	//
	// // Create the tokens and file metadatas in a transaction
	// tx, err := slf.conn.BeginTxx(ctx, nil)
	// if err != nil {
	// 	return err
	// }
	//
	// _, err = tx.NamedExecContext(
	// 	ctx,
	// 	metaCreateStmt,
	// 	metadata,
	// )
	// if err != nil {
	// 	log.Error().Err(err).Msg("failed to update file metadata")
	// 	tx.Rollback()
	// 	return err
	// }
	//
	// _, err = tx.NamedExecContext(
	// 	ctx,
	// 	tokenStmt,
	// 	token,
	// )
	// if err != nil {
	// 	log.Error().Err(err).Msg("failed to create tokens for stream")
	// 	tx.Rollback()
	//
	// 	return err
	// }
	//
	// if err := tx.Commit(); err != nil {
	// 	log.Error().Err(err).Msg("failed to create metadata or tokens")
	// 	return err
	// }
	//
	log.Info().Msg("metadata and tokens created successfully")

	return nil
}
