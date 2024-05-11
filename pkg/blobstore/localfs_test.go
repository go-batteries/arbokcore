package blobstore

import (
	"arbokcore/pkg/utils"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const dirPath = "./tmp/data"

// Test Helpers
func deallocate(t *testing.T, dirPath string) {
	err := os.RemoveAll(dirPath)
	require.NoError(t, err)
}

func Test_EnsureDir(t *testing.T) {
	t.Skip()

	fs, err := NewLocalFS(dirPath)
	require.NoError(t, err)

	defer deallocate(t, fs.dirPath)

	err = EnsureDir(fs.dirPath)
	require.NoError(t, err)

	stat, err := os.Stat(fs.dirPath)
	require.NoError(t, err)

	require.True(t, stat.IsDir())
}

func Test_ReadFileInChunks(t *testing.T) {
	t.Skip()

	t.Run("when file size is multple of 4mb", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(
			context.Background(),
			1*time.Minute,
		)
		defer cancel()

		chunks, err := ReadFileInChunks(ctx, "../../tmp/testdata/24mb.txt")
		require.NoError(t, err)

		require.Equal(t, 6, len(chunks))

	})

	t.Run("when file size is not a multple of 4mb", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(
			context.Background(),
			1*time.Minute,
		)
		defer cancel()

		chunks, err := ReadFileInChunks(ctx, "../../tmp/testdata/33mb.txt")
		require.NoError(t, err)

		require.Equal(t, 9, len(chunks))

	})
}

func Test_WriteFileChunks(t *testing.T) {
	t.Run("file size is a multiple of 4mb chunk size", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			1*time.Minute,
		)
		defer cancel()

		chunks, err := ReadFileInChunks(ctx, "../../tmp/testdata/24mb.txt")
		require.NoError(t, err)

		// require.Equal(t, 1, len(chunks))

		fs, err := NewLocalFS("./tmp/data/")
		require.NoError(t, err)

		defer deallocate(t, fs.dirPath)

		trackerStop := utils.Bench2("ensure dir")
		receivedChunks, err := fs.BatchCreateChunk(ctx, "file4mb", chunks)
		trackerStop()
		require.NoError(t, err)
		//
		require.Equal(t, 1, len(receivedChunks))
		require.Equal(t, len(chunks), len(receivedChunks))
	})

}
