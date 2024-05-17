package blobstore

import (
	"arbokcore/pkg/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

//TODO: use https://pkg.go.dev/github.com/rogpeppe/go-internal/lockedfile to lock the directory

type ChunkedFile struct {
	data    io.Reader
	chunkID int64

	fileDir   string
	chunkPath string
	next      *ChunkedFile
}

const ChunkSize = int64(4 * 1024 * 1024)

func NewChunkedFile(data io.Reader, chunkID int64, next *ChunkedFile) *ChunkedFile {
	return &ChunkedFile{data: data, chunkID: chunkID, next: next}
}

// We could have a BuildFile, because our chunk names are sequential integers,
// ordering isn't much of an issue
// In case of array, the whole file will essentially be in memory
// Instead of loading in parallel, we could load sequentially
// there having a next pointer and a GetNextChunk will help
// Fetch Chunks needs to only load the linkedlist or array
// And then expose another function BuildFile which will take the
// head of the linked list or array, and then append the chunks sequentially
// Since, we will only store structs and not data, until evaluated
// switching to array for better memory access

type BlobStorage interface {
	UpdateChunk(ctx context.Context, fileID string, chunk *ChunkedFile) (string, error)
	BatchCreateChunk(ctx context.Context, fileID string, chunks []*ChunkedFile) ([]*ChunkedFile, error)
	FetchChunks(ctx context.Context, fileID string, chunkIDs []int64) ([]*ChunkedFile, error)
	BuildFile(ctx context.Context, chunkIDs []*ChunkedFile) (io.ReadCloser, error)
}

type LocalFS struct {
	dirPath string
}

var ErrNotDirectory = errors.New("not_a_directory")

func NewLocalFS(dirPath string) (*LocalFS, error) {
	resolvedPath, err := filepath.Abs(dirPath)
	return &LocalFS{dirPath: resolvedPath}, err
}

func EnsureDir(resolvedPath string) error {
	err := os.MkdirAll(resolvedPath, os.ModePerm)
	if err == nil {
		return nil
	}

	if os.IsExist(err) {
		// check if its a directory

		info, err := os.Stat(resolvedPath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			log.Println("exists")
			return nil
		}

		return ErrNotDirectory
	}

	return err
}

func (slf *LocalFS) UpdateChunk(ctx context.Context, fileID string, chunk *ChunkedFile) (string, error) {
	err := EnsureDir(filepath.Join(slf.dirPath, fileID))
	if err != nil {
		return "", err
	}

	path := filepath.Join(slf.dirPath, fileID, fmt.Sprintf("%d", chunk.chunkID))

	tracker := utils.Bench2("file create")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	tracker()

	tracker = utils.Bench2("copy file")
	written, err := io.Copy(file, chunk.data)
	if err != nil {
		return "", nil
	}
	tracker()

	log.Printf("written %d bytes\n", written)

	return path, nil
}

var ErrFileUpload = errors.New("file_upload_failed")

func (slf *LocalFS) BatchCreateChunk(
	ctx context.Context,
	fileID string,
	chunks []*ChunkedFile,
) ([]*ChunkedFile, error) {

	chunksRevceiver := make(chan ChunkedFile, len(chunks))

	// Oh, because the channel closing doesn't happen?
	// Hm
	var wg sync.WaitGroup

	for index, chunk := range chunks {
		wg.Add(1)

		go func(_chunk ChunkedFile, i int64) {
			// TODO: add timeout to ctx
			defer wg.Done()

			filePath, err := slf.UpdateChunk(ctx, fileID, &_chunk)
			if err != nil {
				log.Println(err)
				return
			}

			chunksRevceiver <- ChunkedFile{
				chunkID:   i,
				chunkPath: filePath,
				data:      _chunk.data,
			}
		}(*chunk, int64(index))
	}

	wg.Wait()
	close(chunksRevceiver)

	chunkData := []*ChunkedFile{}

	// Pray that this completes
	for chunk := range chunksRevceiver {
		chunkData = append(chunkData, &chunk)
	}

	if len(chunkData) != len(chunks) {
		return nil, ErrFileUpload
	}

	return chunkData, nil
}

func (slf *LocalFS) FetchChunks(ctx context.Context, fileID string, chunkIDs []int64) ([]*ChunkedFile, error) {
	return nil, nil
}

func (slf *LocalFS) BuildFile(ctx context.Context, chunkIDs []*ChunkedFile) (io.ReadCloser, error) {
	return nil, nil
}
