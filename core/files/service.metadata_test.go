package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestFileInfo(fileID string, chunkID int64) *FilesWithChunks {
	return &FilesWithChunks{
		ID:       fileID,
		Filename: fileID,
		NChunks:  2,
		FileID:   fileID,
		ChunkID:  chunkID,
	}
}

func Test_BuildFilesInfoResponse(t *testing.T) {
	files := []*FilesWithChunks{
		buildTestFileInfo("File1", 0),
		buildTestFileInfo("File1", 1),
		buildTestFileInfo("File2", 0),
		buildTestFileInfo("File2", 1),
	}

	infoResp := BuildFilesInfoResponse(files)

	require.Equal(t, len(infoResp), 2)

	require.Equal(t, len(infoResp[0].Chunks), 2)
	require.Equal(t, len(infoResp[1].Chunks), 2)

	assert.Equal(t, infoResp[0].Name, "File1")
	assert.Equal(t, infoResp[1].Name, "File2")
}
