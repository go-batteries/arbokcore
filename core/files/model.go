package files

import (
	"arbokcore/core/database"
)

const (
	StatusUploading = "uploading"
	StatusFailed    = "failed"
	StatusCompleted = "completed"
)

const FrontendChunkSize int64 = 4 * 1024 * 1024

type FileMetadata struct {
	ID       string `db:"id"`
	UserID   string `db:"user_id"`
	Filename string `db:"file_name"`
	FileSize int64  `db:"file_size"`
	FileType string `db:"file_type"`
	FileHash string `db:"file_hash"`
	NChunks  int    `db:"chunks"` // In MB

	UploadStaus string `db:"upload_status"`
	database.Timestamp
}

type UserFile struct {
	UserID       string `db:"user_id"`
	FileID       string `db:"file_id"`
	ChunkID      int64  `db:"chunk_id"`
	ChunkBlobUrl string `db:"chunk_blob_url"`
	ChunkHash    string `db:"chunk_hash"`
	NextChunkID  *int64 `db:"next_chunk_id"`
	Version      int    `db:"version"`

	database.Timestamp
}

type FilesWithChunks struct {
	ID       string `db:"id" json:"fileID"`
	UserID   string `db:"user_id" json:"-"`
	Filename string `db:"file_name" json:"fileName"`
	FileSize int64  `db:"file_size" json:"fileSize"`
	FileType string `db:"file_type" json:"fileType"`
	FileHash string `db:"file_hash" json:"fileHash"`
	NChunks  int    `db:"chunks" json:"chunks"` // In MB

	UploadStaus  string `db:"upload_status" json:"uploadStatus"`
	FileID       string `db:"file_id" json:"-"`
	ChunkID      int64  `db:"chunk_id" json:"chunkID"`
	ChunkBlobUrl string `db:"chunk_blob_url" json:"chunkBlobUrl"`
	ChunkHash    string `db:"chunk_hash" json:"chunkHash"`
	NextChunkID  *int64 `db:"next_chunk_id" json:"nextChunkID"`
	Version      int    `db:"version" json:"version"`

	database.Timestamp
}

func CalculateChunks(fileSize int64) int {
	n_chunks := float64(fileSize) / float64(FrontendChunkSize)

	if float64(int(n_chunks)) < n_chunks {
		n_chunks += 1.0
	}

	return int(n_chunks)

}
