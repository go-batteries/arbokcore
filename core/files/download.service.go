package files

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type DownloadHandler struct {
	cacher *redis.Client
}

func NewDownloadHandler(client *redis.Client) *DownloadHandler {
	return &DownloadHandler{cacher: client}
}

func (dh *DownloadHandler) Download(ctx context.Context, fileSize int64, filePaths []string) (*bytes.Buffer, error) {

	var buffer = bytes.NewBuffer(make([]byte, 0, fileSize))

	for _, filePath := range filePaths {
		data, err := dh.cacher.Get(ctx, filePath).Bytes()
		if err == nil {
			w, err := io.Copy(buffer, bytes.NewBuffer(data))
			if err != nil {
				return nil, err
			}

			log.Info().Int64("bytes", w).Msg("written chunks")
			continue
		}

		//TODO: replace with interface, so s3/os all have same interface
		file, err := os.Open(filePath)
		if err != nil {
			log.Error().Err(err).Msg("failed to read intentend file")
			return nil, err
		}

		defer file.Close()

		fileDataBuffer := new(bytes.Buffer)
		teeReader := io.TeeReader(file, fileDataBuffer)

		w, err := io.Copy(buffer, teeReader)
		if err != nil {
			log.Error().Err(err).Msg("failed to copy file chunk into buffer")
			return nil, err
		}

		log.Info().Int64("bytes", w).Msg("written chunks")
		err = dh.cacher.SetEx(ctx, filePath, fileDataBuffer.Bytes(), 2*time.Hour).Err()
		if err != nil {
			log.Error().Err(err).Msg("failed to set data in cache")
		}
	}

	return buffer, nil
}
