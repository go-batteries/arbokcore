package routes

import (
	"arbokcore/core/api"
	"arbokcore/core/files"
	"arbokcore/core/tokens"
	"arbokcore/web/middlewares"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type MetadataHandler struct {
	FileSvc *files.MetadataService
}

const (
	RouteMetadataGet    = "/my/files"
	RouteMetadataPost   = "/my/files"
	RouteMetadataPatch  = "/my/files/:fileID"
	RouteFileUploadDone = "/my/files/:fileID/eof"
)

func (handler *MetadataHandler) MarkUploadComplete(c echo.Context) error {
	token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
	if !ok {
		log.Error().Msg("token validation not done")
		return c.NoContent(http.StatusUnauthorized)
	}

	ctx := c.Request().Context()

	if token.ResourceID != c.Param("fileID") {
		log.Error().
			Str("resource_id", token.ResourceID).
			Str("fileID", c.Param("fileID")).
			Msg("fileID for the stream token invalid")

		return c.NoContent(http.StatusBadRequest)
	}

	resp := handler.FileSvc.MarkUploadComplete(ctx, token.ResourceID)
	if !resp.Success {
		return c.JSON(resp.Error.HttpStatus, resp)
	}

	return c.JSON(http.StatusOK, resp)

}

func (handler *MetadataHandler) DownloadFile(c echo.Context) error {
	token, ok := c.Get(middlewares.TokenContextKey).(*tokens.Token)
	if !ok {
		log.Error().Msg("token validation not done")
		return c.NoContent(http.StatusUnauthorized)
	}

	ctx := c.Request().Context()

	fileID := c.Param("fileID")
	downloadUrls, infoResp, err := handler.FileSvc.ListOrderedFileChunks(ctx, fileID, *token.UserID)

	if err != nil {
		log.Error().Err(err).Msg("failed to get files chunks")
		resp := api.BuildResponse(err, nil)
		return c.JSON(resp.Error.HttpStatus, resp)
	}

	var buffer = bytes.NewBuffer(make([]byte, 0, infoResp.Size))

	for _, filePath := range downloadUrls {
		file, err := os.Open(filePath)
		if err != nil {
			log.Error().Err(err).Msg("failed to find file in url")
			return c.NoContent(http.StatusInternalServerError)
		}

		defer file.Close()

		w, err := io.Copy(buffer, file)
		if err != nil {
			log.Error().Err(err).Msg("failed to copy file chunk into buffer")
			return c.NoContent(http.StatusInternalServerError)
		}

		log.Info().Int64("bytes", w).Msg("written chunks")
	}

	fmt.Println("total size ", buffer.Len())

	outputFilePath := fmt.Sprintf("./tmp/outdir/%s", infoResp.Name)
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		log.Error().Err(err).Msg("failed to create output file")
		return c.NoContent(http.StatusInternalServerError)
	}

	defer outputFile.Close()
	defer os.Remove(outputFilePath)

	_, err = io.Copy(outputFile, buffer)
	if err != nil {
		log.Error().Err(err).Msg("failed to write to output file")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.Attachment(outputFilePath, infoResp.Name)
}

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
		UserID:       token.ResourceID,
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

	resp := handler.FileSvc.ListFilesForUser(ctx, token.ResourceID, offset)

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

	req.UserID = token.ResourceID

	resp := handler.FileSvc.UpdateFileMetadata(ctx, req)
	if resp.Success {
		return c.JSON(http.StatusOK, resp)
	}

	return c.JSON(resp.Error.HttpStatus, resp)
}
