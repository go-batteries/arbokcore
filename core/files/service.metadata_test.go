package files

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestFileInfo(fileID string, chunkID int64, version string) *FilesWithChunks {
	return &FilesWithChunks{
		ID:       fileID,
		Filename: fileID,
		NChunks:  2,
		FileID:   fileID,
		ChunkID:  chunkID,
	}
}

func Test_BuildFilesInfoResponse(t *testing.T) {
	t.Run("different files for a given version", func(t *testing.T) {

		files := []*FilesWithChunks{
			buildTestFileInfo("File1", 0, "1"),
			buildTestFileInfo("File1", 1, "1"),
			buildTestFileInfo("File2", 0, "1"),
			buildTestFileInfo("File2", 1, "1"),
		}

		infoResp := BuildFilesInfoResponse(files)
		log.Printf("%+v\n", infoResp[1])

		require.Equal(t, len(infoResp), 2)

		require.Equal(t, len(infoResp[0].VersionChunks["1"]), 2)
		require.Equal(t, len(infoResp[1].VersionChunks["1"]), 2)

		assert.Equal(t, infoResp[0].Name, "File1")
		assert.Equal(t, infoResp[1].Name, "File2")
	})

	t.Run("different versions for the same file", func(t *testing.T) {
		files := []*FilesWithChunks{
			buildTestFileInfo("File1", 0, "1"),
			buildTestFileInfo("File1", 1, "1"),
			buildTestFileInfo("File1", 0, "2"),
			buildTestFileInfo("File1", 1, "2"),
		}

		infoResp := BuildFilesInfoResponse(files)

		require.Equal(t, len(infoResp), 1)
		require.Equal(t, len(infoResp[0].VersionChunks), 2)
	})
}
