package tokens

import (
	"arbokcore/pkg/squirtle"
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type FindByClause struct {
	AccessToken  string
	RefreshToken string
	StreamToken  string

	ResourceID   string
	ResourceType string
	UserID       *string

	TestMode bool
}

type Repository interface {
	CreateSession(ctx context.Context, resourceID, resourceType string, userID *string) (*Token, error)
	FindByAccessToken(ctx context.Context, clause FindByClause) (*Token, error)
	FindByStreamToken(ctx context.Context, clause FindByClause) (*Token, error)
}

type TokensRepository struct {
	conn    *sqlx.DB
	querier squirtle.QueryMapper
}

func NewTokensRepository(
	conn *sqlx.DB,
	querier squirtle.QueryMapper,
) *TokensRepository {

	return &TokensRepository{conn: conn, querier: querier}
}

const (
	CreateTokenStmt    = "CreateToken"
	GetAccessTokenStmt = "GetAccessToken"
	GetStreamTokenStmt = "GetStreamToken"
)

var (
	ErrTokenNotFound    = errors.New("token_not_found")
	ErrTokenExpired     = errors.New("token_expired:2007:410")
	ErrStatmentNotFound = errors.New("stmt_not_found:2008:500")
)

func (tsrepo *TokensRepository) FindByAccessToken(
	ctx context.Context,
	clause FindByClause,
) (*Token, error) {

	if clause.TestMode {
		log.Info().Msg("authorization in test mode")

		return tsrepo.findByTestMode(ctx, clause)
	}

	stmt, ok := tsrepo.querier.GetQuery(GetStreamTokenStmt)
	if !ok {
		return nil, ErrStatmentNotFound
	}

	token := &Token{}

	rows, err := tsrepo.conn.NamedQueryContext(
		ctx,
		stmt,
		map[string]any{
			"access_token":  clause.AccessToken,
			"resource_id":   clause.ResourceID, //user_id
			"resource_type": "user",
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to get session access token")
		return nil, err
	}

	for rows.Next() {
		err = rows.StructScan(token)
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshall access token")
		return nil, err
	}

	if token.HasAccessExpired() {
		log.Error().Msg("access token expired")
		return nil, ErrTokenExpired
	}

	return token, nil
}

func (tsrepo *TokensRepository) findByTestMode(
	ctx context.Context,
	clause FindByClause,
) (*Token, error) {

	_ = ctx

	// if clause.ResourceType == "stream" {
	// 	token = &StreamToken
	//
	// 	if token.AccessToken != clause.StreamToken {
	// 		return nil, ErrTokenNotFound
	// 	}
	// }

	allowedTokens := []*Token{&AdminToken, &AnotherToken}

	for _, token := range allowedTokens {
		if token.IsExpired() {
			return nil, ErrTokenExpired
		}

		if token.AccessToken == clause.AccessToken {
			return token, nil
		}
	}

	return nil, ErrTokenNotFound
}

func (tsrepo *TokensRepository) FindByStreamToken(
	ctx context.Context,
	clause FindByClause,
) (*Token, error) {

	stmt, ok := tsrepo.querier.GetQuery(GetStreamTokenStmt)
	if !ok {
		return nil, ErrStatmentNotFound
	}

	token := &Token{}

	// log.Info().Msgf("token clause %+v", clause)
	rows, err := tsrepo.conn.NamedQueryContext(
		ctx,
		stmt,
		map[string]any{
			"access_token":  clause.StreamToken,
			"resource_id":   clause.ResourceID, //file_id
			"resource_type": "stream",
			"user_id":       clause.UserID,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to get stream access token")
		return nil, err
	}

	for rows.Next() {
		err = rows.StructScan(token)
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to destructure stream access token")
		return nil, err
	}

	if token.HasAccessExpired() {
		log.Error().Msg("stream token expired")
		return nil, ErrTokenExpired
	}

	return token, nil
}

func (tsrepo *TokensRepository) CreateSession(
	ctx context.Context,
	resourceID string,
	resourceType string,
	userID *string,
) (*Token, error) {

	token, err := NewToken(
		resourceID, resourceType, // We will get rid of this later
		WithUserID(userID),
		WithDefaultExpiry(),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to build token")
		return nil, errors.New("token_create_failed:3001:500")
	}

	stmt, ok := tsrepo.querier.GetQuery(CreateTokenStmt)
	if !ok {
		return nil, ErrStatmentNotFound
	}

	_, err = tsrepo.conn.NamedExecContext(ctx, stmt, token)
	if err != nil {
		log.Error().Err(err).Msg("failed to create token")
		return nil, errors.New("token_create_failed:3002:500")
	}

	return token, nil
}
