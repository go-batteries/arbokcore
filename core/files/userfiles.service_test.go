package files

import (
	"arbokcore/core/api"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ValidateHashDigest(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("./README.md")
	require.NoError(t, err, "should not have failed to find file")

	req := &api.FileChunkRequest{
		Data:        f,
		ChunkDigest: "f1f8b2de17e1ea39774d8b9f6893e0e421593505414bff84a4f4365881c71f04",
	}

	err = ValidateHash(ctx, req)
	require.NoError(t, err)
}
