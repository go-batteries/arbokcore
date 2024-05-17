package supervisors

import (
	"arbokcore/core/files"
	"arbokcore/pkg/queuer"
	"arbokcore/pkg/rho"
	"arbokcore/pkg/workerpool"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Execute(ctx context.Context, data []*queuer.Payload) []*queuer.Payload {
	return data
}

type MockSuperVisor struct {
	demand chan int
}

func (slf *MockSuperVisor) Produce(ctx context.Context) chan []*queuer.Payload {
	resultsCh := make(chan []*queuer.Payload, 1)

	go func() {
		defer close(resultsCh)

		for {
			select {

			case d := <-slf.demand:
				results := []*queuer.Payload{}

				for i := 0; i < d; i++ {
					payload := &queuer.Payload{
						Message: []byte(fmt.Sprintf("%d", i)),
					}
					results = append(results, payload)
				}

				//Mock sleep to give other workers
				// a chance to pickup produced data
				time.Sleep(2 * time.Second)
				resultsCh <- results

			case <-ctx.Done():
				return
			}
		}
	}()

	return resultsCh
}

func (slf *MockSuperVisor) Demand(val int) {
	slf.demand <- val
}

var thisFileMetaStr = `
{
  "fileID": "01HY38Y6X2GHW56X39N2Y2T8HQ",
  "fileName": "An Overview of the LlamaIndex Framework by cbarkinozer Medium.pdf",
  "fileHash": "8b134c72cfc823dd8ab609fc298808e142304c54fe6bbef6ad0a12a91e6f7ab9",
  "fileSize": 6559150,
  "fileType": "application/pdf",
  "currentFlag": true,
  "chunks": {
    "1": {
      "fileID": "01HY38Y6X2GHW56X39N2Y2T8HQ",
      "fileName": "An Overview of the LlamaIndex Framework by cbarkinozer Medium.pdf",
      "fileSize": 6559150,
      "fileType": "application/pdf",
      "fileHash": "8b134c72cfc823dd8ab609fc298808e142304c54fe6bbef6ad0a12a91e6f7ab9",
      "chunks": 2,
      "currentFlag": true,
      "uploadStatus": "",
      "chunkID": 1,
      "chunkBlobUrl": "/Users/alexday/dev/gogogo/fileshhh/core/tmp/arbokdata/01HY38Y6X2GHW56X39N2Y2T8HQ/1",
      "chunkHash": "e0d865aa9106752782c4078e31bd05338d2ec3531e175e2a58abadea472482a2",
      "nextChunkID": -1,
      "prevID": "01HY38X7WT2NS3Z130FXX930E4",
      "EndDate": null,
      "createdAt": "2024-05-17T12:37:56.801093Z",
      "updatedAt": "2024-05-17T12:37:56.801093Z"
    }
  },
  "nChunks": 2,
  "userID": "U11223455"
}`

var prevFileMetaStr = `
{
  "fileID": "01HY38X7WT2NS3Z130FXX930E4",
  "fileName": "An Overview of the LlamaIndex Framework by cbarkinozer Medium.pdf",
  "fileHash": "9c0f65355c79c15ccfb55706a70b930a06286ddfc115b0314ee5e37566d2a6c3",
  "fileSize": 6522997,
  "fileType": "application/pdf",
  "currentFlag": false,
  "chunks": {
    "0": {
      "fileID": "01HY38X7WT2NS3Z130FXX930E4",
      "fileName": "An Overview of the LlamaIndex Framework by cbarkinozer Medium.pdf",
      "fileSize": 6522997,
      "fileType": "application/pdf",
      "fileHash": "9c0f65355c79c15ccfb55706a70b930a06286ddfc115b0314ee5e37566d2a6c3",
      "chunks": 2,
      "currentFlag": false,
      "uploadStatus": "",
      "chunkID": 0,
      "chunkBlobUrl": "/Users/alexday/dev/gogogo/fileshhh/core/tmp/arbokdata/01HY38X7WT2NS3Z130FXX930E4/0",
      "chunkHash": "04eb9fc517e6937ff3dc10542e89b666af2a7b8c58cfff3c9704ded8114deb8c",
      "nextChunkID": 1,
      "prevID": null,
      "EndDate": "2024-05-17T12:37:56.805944Z",
      "createdAt": "2024-05-17T12:37:25.053663Z",
      "updatedAt": "2024-05-17T12:37:25.053663Z"
    },
    "1": {
      "fileID": "01HY38X7WT2NS3Z130FXX930E4",
      "fileName": "An Overview of the LlamaIndex Framework by cbarkinozer Medium.pdf",
      "fileSize": 6522997,
      "fileType": "application/pdf",
      "fileHash": "9c0f65355c79c15ccfb55706a70b930a06286ddfc115b0314ee5e37566d2a6c3",
      "chunks": 2,
      "currentFlag": false,
      "uploadStatus": "",
      "chunkID": 1,
      "chunkBlobUrl": "/Users/alexday/dev/gogogo/fileshhh/core/tmp/arbokdata/01HY38X7WT2NS3Z130FXX930E4/1",
      "chunkHash": "c9e316246d034c5820e4ddf525fc3736305ae586ca2b77ea8a94f4b222db6198",
      "nextChunkID": -1,
      "prevID": null,
      "EndDate": "2024-05-17T12:37:56.805944Z",
      "createdAt": "2024-05-17T12:37:25.05217Z",
      "updatedAt": "2024-05-17T12:37:25.05217Z"
    }
  },
  "nChunks": 2,
  "userID": "U11223455"
}`

func Test_Supervisor(t *testing.T) {
	t.Skip()

	ctx := context.Background()
	pool := workerpool.NewWorkerPool(2, Execute)
	outChan := make(chan []*queuer.Payload, 1)
	demand := make(chan int, 1)

	ms := MockSuperVisor{demand: demand}

	receivCh := ms.Produce(ctx)
	go workerpool.Dispatch(ctx, pool, receivCh)

	pool.Start(ctx)

	workerpool.Merge(ctx, pool.ResultChs, outChan)

	ms.Demand(10)
	ms.Demand(10)

	payloads := <-outChan
	payloads = append(payloads, <-outChan...)

	pool.Stop(ctx)

	msg := rho.Map(
		payloads,
		func(payload *queuer.Payload, _ int) string {
			return string(payload.Message)
		})

	expected := []string{}
	for i := 0; i < 10; i++ {
		expected = append(expected, fmt.Sprintf("%d", i))
	}

	expected = append(expected, expected...)

	require.Equal(t, expected, msg)
}

func Test_RestOfChunks(t *testing.T) {
	thisFile := &files.FileInfoResponse{}

	err := json.Unmarshal([]byte(thisFileMetaStr), thisFile)
	require.NoError(t, err)

	prevFile := &files.FileInfoResponse{}

	err = json.Unmarshal([]byte(prevFileMetaStr), prevFile)
	require.NoError(t, err)

	chunks := RestOfChunks(thisFile, prevFile)

	require.Equal(t, len(chunks), 1)
}

func Test_ReconstructChunks(t *testing.T) {
	thisFile := &files.FileInfoResponse{}

	err := json.Unmarshal([]byte(thisFileMetaStr), thisFile)
	require.NoError(t, err)
	require.Equal(t, 1, len(thisFile.Chunks))

	keys := []string{}
	for id := range thisFile.Chunks {
		keys = append(keys, id)
	}
	require.ElementsMatch(t, []string{"1"}, keys)

	prevFile := &files.FileInfoResponse{}

	err = json.Unmarshal([]byte(prevFileMetaStr), prevFile)
	require.NoError(t, err)
	require.Equal(t, 2, len(prevFile.Chunks))

	keys = []string{}
	for id := range prevFile.Chunks {
		keys = append(keys, id)
	}
	require.ElementsMatch(t, []string{"0", "1"}, keys)

	response := ReconstructChunks(thisFile, prevFile)
	require.Equal(t, 2, len(response.Chunks))

	keys = []string{}
	for id := range response.Chunks {
		keys = append(keys, id)
	}
	require.ElementsMatch(t, []string{"0", "1"}, keys)
}

func toPtr[E any](v E) *E {
	return &v
}

func Test_Validate(t *testing.T) {
	t.Run("when new chunk chain matches prev valid chunk chain", func(t *testing.T) {

		rc := &files.FileInfoResponse{
			NChunks: 3,
			Chunks: map[string]*files.FilesWithChunks{
				"0": {NextChunkID: toPtr(int64(1)), ChunkHash: "a"},
				"1": {NextChunkID: toPtr(int64(2)), ChunkHash: "b"},
				"2": {NextChunkID: toPtr(int64(-1)), ChunkHash: "c"},
			},
		}

		pc := &files.FileInfoResponse{
			NChunks: 3,
			Chunks: map[string]*files.FilesWithChunks{
				"0": {NextChunkID: toPtr(int64(1)), ChunkHash: "a"},
				"1": {NextChunkID: toPtr(int64(2)), ChunkHash: "d"},
				"2": {NextChunkID: toPtr(int64(-1)), ChunkHash: "c"},
			},
		}

		err := Validate(rc, pc)
		require.True(t, err)
	})

	t.Run("when new chunk chain matches doesnot match previous chain", func(t *testing.T) {

		rc := &files.FileInfoResponse{
			NChunks: 3,
			Chunks: map[string]*files.FilesWithChunks{
				"0": {NextChunkID: toPtr(int64(2)), ChunkHash: "a"},
				"1": {NextChunkID: toPtr(int64(-1)), ChunkHash: "b"},
				"2": {NextChunkID: toPtr(int64(1)), ChunkHash: "c"},
			},
		}

		pc := &files.FileInfoResponse{
			NChunks: 3,
			Chunks: map[string]*files.FilesWithChunks{
				"0": {NextChunkID: toPtr(int64(1)), ChunkHash: "a"},
				"1": {NextChunkID: toPtr(int64(2)), ChunkHash: "b"},
				"2": {NextChunkID: toPtr(int64(-1)), ChunkHash: "c"},
			},
		}

		err := Validate(rc, pc)
		require.False(t, err)
	})
}
