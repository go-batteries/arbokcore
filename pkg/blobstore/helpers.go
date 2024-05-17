package blobstore

import (
	"arbokcore/pkg/utils"
	"context"
	"time"
)

func ReadFileInChunks(ctx context.Context, filePath string) (chunks []*ChunkedFile, err error) {
	utils.Bench(time.Now(), "ReadFileInChunks")

	// file, err := os.Open(filePath)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// info, err := file.Stat()
	// if err != nil {
	// 	return nil, err
	// }
	//
	// fileSize := info.Size()
	// numChunks := (fileSize + ChunkSize - 1) / ChunkSize
	//
	// chunks := []*ChunkedFile{}
	// reader := io.NewSectionReader(file, 0, fileSize)
	//
	// // log.Println(fileSize, numChunks)
	//
	// for i := 0; i < int(numChunks)-1; i++ {
	// 	byts := make([]byte, ChunkSize)
	//
	// 	n, err := reader.Read(byts)
	// 	if err != nil {
	// 		// log.Printf("e1")
	// 		return nil, err
	// 	}
	//
	// 	_ = n
	//
	// 	chunks = append(chunks, &ChunkedFile{
	// 		chunkID: int64(i),
	// 		data:    bytes.NewReader(byts),
	// 	})
	// }
	//
	// rest, err := io.ReadAll(reader)
	// if err != nil {
	// 	// log.Printf("e2")
	// 	return nil, err
	// }
	//
	// chunks = append(chunks, &ChunkedFile{
	// 	chunkID: numChunks - 1,
	// 	data:    bytes.NewReader(rest),
	// })
	//
	return chunks, nil
}
