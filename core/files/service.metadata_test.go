package files

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func buildTestFileInfo(fileID string, chunkID int64, hash string, prevID *string) *FilesWithChunks {
	return &FilesWithChunks{
		ID:        fileID,
		Filename:  fileID,
		FileID:    fileID,
		ChunkID:   chunkID,
		ChunkHash: hash,
		PrevID:    prevID,
	}
}

func Test_BuildFilesInfoResponse(t *testing.T) {
	t.Run("new file with multiple chunks", func(t *testing.T) {

		files := []*FilesWithChunks{
			buildTestFileInfo("File1", 0, "F1", nil),
			buildTestFileInfo("File1", 1, "F2", nil),
		}

		infoResp := BuildFilesInfoResponse(files)
		require.Equal(t, len(infoResp), 1)

		expectedResp := []*FileInfoResponse{
			&FileInfoResponse{
				ID:   "File1",
				Name: "File1",
				Chunks: map[string]*FilesWithChunks{
					"0": files[0],
					"1": files[1],
				},
			},
		}

		require.Equal(t, expectedResp, infoResp)
	})

	t.Run("same files with previous versions", func(t *testing.T) {
		prevID := "File1"

		files := []*FilesWithChunks{
			buildTestFileInfo("File1", 0, "F1", nil),
			buildTestFileInfo("File1", 1, "F2", nil),

			// 0th Chunk of File changed
			buildTestFileInfo("File11", 0, "E1", &prevID),
			buildTestFileInfo("File11", 1, "F2", &prevID),
		}

		infoResp := BuildFilesInfoResponse(files)

		// Its one file with 2 chunks
		require.Equal(t, len(infoResp), 2)

		expectedResp := []*FileInfoResponse{
			&FileInfoResponse{
				ID:   "File1",
				Name: "File1",
				Chunks: map[string]*FilesWithChunks{
					"0": files[0],
					"1": files[1],
				},
			},
			&FileInfoResponse{
				ID:   "File11",
				Name: "File11",
				Chunks: map[string]*FilesWithChunks{
					"0": files[2],
					"1": files[3],
				},
			},
		}

		require.Equal(t, len(expectedResp), len(infoResp))
		require.Equal(t, expectedResp, infoResp)

	})
}
