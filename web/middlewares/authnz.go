package middlewares

import (
	"arbokcore/core/tokens"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

const (
	AccessTokenHeaderKey = "X-Access-Token"
	StreamTokenHeaderKey = "X-Stream-Token"

	BearerKey = "Bearer "
	BasicKey  = "Basic "

	TokenContextKey = "current_token"
)

type AuthMidllewareService struct {
	repo tokens.Repository
}

func NewAuthMiddleWareService(repo tokens.Repository) *AuthMidllewareService {
	return &AuthMidllewareService{repo: repo}
}

func getBearerToken(token string) (string, bool) {
	if !strings.HasPrefix(token, BearerKey) {
		return "", false
	}

	return strings.TrimSpace(strings.TrimPrefix(token, BearerKey)), true
}

func (slf *AuthMidllewareService) ValidateAccessToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log.Info().Msg("validating access token")

		headers := c.Request().Header
		accessTokenHeader := headers.Get(AccessTokenHeaderKey)

		if accessTokenHeader == "" {
			return c.NoContent(http.StatusUnauthorized)
		}

		accessToken, ok := getBearerToken(accessTokenHeader)
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}

		ctx := c.Request().Context()

		token, err := slf.repo.FindByAccessToken(ctx, tokens.FindByClause{
			ResourceType: "user",
			AccessToken:  accessToken,
			TestMode:     true,
		})

		if err != nil {
			log.Error().Err(err).Msg("failed to validate access token")
			return c.NoContent(http.StatusUnauthorized)
		}

		log.Info().Msg("access token validation success")

		c.Set(TokenContextKey, token)

		return next(c)

	}
}

func (slf *AuthMidllewareService) ValidateStreamToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		headers := c.Request().Header
		accessTokenHeader := headers.Get(AccessTokenHeaderKey)
		streamTokenHeader := headers.Get(StreamTokenHeaderKey)

		if accessTokenHeader == "" || streamTokenHeader == "" {
			log.Error().
				Str("accesstoken", accessTokenHeader).
				Str("streamtoken", streamTokenHeader).
				Msg("access or stream token in header missing")

			return c.NoContent(http.StatusUnauthorized)
		}

		accessToken, ok := getBearerToken(accessTokenHeader)
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}

		streamToken, ok := getBearerToken(streamTokenHeader)
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}

		log.Info().Msg("validating stream token")

		fileID := c.QueryParam("fileID")
		if fileID == "" {
			fileID = c.Param("fileID")
		}

		ctx := c.Request().Context()
		token, err := slf.repo.FindByStreamToken(ctx, tokens.FindByClause{
			ResourceType: "stream",
			ResourceID:   fileID,
			AccessToken:  accessToken,
			TestMode:     true,
			StreamToken:  streamToken,
			UserID:       &tokens.AdminToken.ResourceID, // TODO: get userID from access token
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to validate stream token")
			return c.NoContent(http.StatusUnauthorized)
		}

		c.Set(TokenContextKey, token)

		return next(c)

	}
}
