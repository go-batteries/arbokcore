package routes

import (
	"arbokcore/core/api"
	"arbokcore/core/files"
	"arbokcore/core/tokens"
	"arbokcore/web/middlewares"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type ChunkHandler struct {
	ChunkSvc *files.FileChunkService
}

const (
	RouteChunkUpdate = "/my/files/:fileID/chunks"
)

func (handler *ChunkHandler) UpsertChunks(c echo.Context) error {
	req := &api.FileChunkRequest{}

	ctx := c.Request().Context()

	token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
	if !ok {
		log.Error().Msg("current token is not set")
		return c.NoContent(http.StatusForbidden)
	}

	if token.UserID == nil {
		log.Error().Msg("userid is not found for token")
		return c.NoContent(http.StatusForbidden)
	}

	if err := c.Bind(req); err != nil {
		log.Error().Err(err).Msg("failed to parse form request")
		return c.NoContent(http.StatusInternalServerError)
	}

	fh, err := c.FormFile("data")
	if err != nil {
		log.Error().Err(err).Msg("failed to read chunk from form")
		return c.NoContent(http.StatusBadRequest)
	}

	file, err := fh.Open()
	if err != nil {
		log.Error().Err(err).Msg("failed to open file chunk")
		return c.NoContent(http.StatusInternalServerError)
	}

	// fmt.Printf("chunk %+v, size %d\n", req, fh.Size)

	req.Data = file
	req.FileID = token.ResouceID
	req.UserID = *token.UserID

	resp := handler.ChunkSvc.Create(ctx, req)

	// fmt.Printf("respp %+v\n", resp)

	if resp.Success {
		return c.JSON(http.StatusCreated, resp)
	}

	return c.JSON(resp.Error.HttpStatus, resp)
}
