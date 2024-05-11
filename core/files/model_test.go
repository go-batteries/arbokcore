package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CalculateChunks(t *testing.T) {
	t.Run("when chunk size is divisble by 4 return even chunk size", func(t *testing.T) {
		chunkSize := 8 * 1024 * 1024

		assert.Equal(t, 2, CalculateChunks(int64(chunkSize)))
	})

	t.Run("when chunk size is not divisble by 4 return 1 + chunk size", func(t *testing.T) {
		chunkSize := 9 * 1024 * 1024

		assert.Equal(t, 3, CalculateChunks(int64(chunkSize)))
	})

	t.Run("when chunk size is not divisble by 4, but even return 1 + chunk size", func(t *testing.T) {
		chunkSize := 10 * 1024 * 1024

		assert.Equal(t, 3, CalculateChunks(int64(chunkSize)))
	})
}
