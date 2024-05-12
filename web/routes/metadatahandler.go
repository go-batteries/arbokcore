package routes

import (
	"arbokcore/core/api"
	"arbokcore/core/files"
	"arbokcore/core/tokens"
	"arbokcore/web/middlewares"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type MetadataHandler struct {
	FileSvc *files.MetadataService
}

const (
	RouteMetadataGet   = "/my/files"
	RouteMetadataPost  = "/my/files"
	RouteMetadataPatch = "/my/files/:fileID"
)

func (handler *MetadataHandler) PostFileMetadata(c echo.Context) error {
	req := &files.MetadataRequest{}

	if err := c.Bind(req); err != nil {
		log.Error().Err(err).Msg("bad request")
		return c.NoContent(http.StatusBadRequest)
	}
	// Get the Reuest JSON body
	// Call the Service method to create record

	ctx := c.Request().Context()
	token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
	if !ok {
		log.Error().Msg("token validation not done")
		return c.NoContent(http.StatusUnauthorized)
	}

	resp := handler.FileSvc.PrepareFileForUpload(ctx, files.MetadataRequest{
		UserID:       token.ResouceID,
		FileName:     req.FileName,
		FileType:     req.FileType,
		FileSize:     req.FileSize,
		Digest:       req.Digest,
		Chunks:       req.Chunks,
		UploadStatus: files.StatusUploading,
	})

	if resp.Success {
		data, ok := resp.Data.(*files.MetadataTokenResponse)
		_ = ok
		WriteSessionCookie(c, int(data.ExpiresIn), "stream_token", data.StreamToken)

		return c.JSON(http.StatusCreated, resp)
	}

	return c.JSON(resp.Error.HttpStatus, resp)
}

func WriteSessionCookie(c echo.Context, maxAge int, key string, value string) {
	log.Info().Msg("setting value in cookie")

	cookie := &http.Cookie{
		Name:     key,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   maxAge,
		SameSite: http.SameSiteNoneMode,
	}

	c.SetCookie(cookie)
}

func (handler *MetadataHandler) GetFileMetadata(c echo.Context) error {
	ctx := c.Request().Context()

	token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
	if !ok {
		log.Error().Msg("current token is not set")
		return c.NoContent(http.StatusForbidden)
	}

	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil {
		offset = 0
	}

	resp := handler.FileSvc.ListFilesForUser(ctx, token.ResouceID, offset)

	if resp.Success {
		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(resp.Error.HttpStatus, resp)
}

func (handler *MetadataHandler) UpdateFileMetadata(c echo.Context) error {
	req := &api.FileUpdateMetadataRequest{}

	if err := c.Bind(req); err != nil {
		log.Error().Err(err).Msg("failed to parse request")
		return c.NoContent(http.StatusBadRequest)
	}

	ctx := c.Request().Context()

	//TODO: validate if user has file access
	//TODO: validate request struct
	token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
	if !ok {
		log.Error().Msg("current token is not set")
		return c.NoContent(http.StatusForbidden)
	}
	req.UserID = token.ResouceID

	resp := handler.FileSvc.UpdateFileMetadata(ctx, req)
	if resp.Success {
		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(resp.Error.HttpStatus, resp)
}