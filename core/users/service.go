package users

import (
	"context"
	"errors"

	"arbokcore/core/tokens"
	"arbokcore/pkg/squirtle"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type Session struct {
}

const (
	CreateUserQueryKey = "CreateUserQuery"
	SelectUserQueryKey = "GetUserByEmail"
)

type Service interface {
	Create(ctx context.Context, email, password string) (*Session, error)
}

type UserService struct {
	conn    *sqlx.DB
	querier squirtle.QueryMapper
}

func NewUserService(conn *sqlx.DB, querier squirtle.QueryMapper) *UserService {
	return &UserService{conn: conn, querier: querier}
}

var (
	UserCreationFailed = errors.New("user_creation_failed:1001:500")
)

func (us *UserService) Create(ctx context.Context, email, password string) (*tokens.Token, error) {
	user, err := NewUser(email, password)
	if err != nil {
		log.Error().Err(err).Msg("failed to build user")
		return nil, UserCreationFailed
	}

	stmt, ok := us.querier.GetQuery(CreateUserQueryKey)
	if !ok {
		log.Error().Msg("failed to get query")
		return nil, errors.New("query_retriever_failed:1002:500")
	}

	_, err = us.conn.NamedExecContext(ctx, stmt, user)
	if err != nil {
		log.Error().Err(err).Msg("failed to insert users into database")
	}

	// TODO: Create tokens and allow login
	return &tokens.AdminToken, nil
}
