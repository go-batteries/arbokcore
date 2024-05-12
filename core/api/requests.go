package api

import "io"

// I have made a change here
// Moved the api request structure
// From web/ to core/api
// Reason is to prevent circular dependency
// The service layers receive the request object
// And constructs the model object which is sent to
// Repository layer
// I took this directory split from the
// auth0's golang example
// But they have a problem
// There the problem is, the controller now needs
// to know about the model
// I don't know, it's about personal preference I guess
// But to me, I don't think models and repository layer
// are coupled. And service is doing the validation
// and other business logic

// If I put the model struct in web/ layer, these
// logics will start leaking,
// Controllers should do controller level validations
// Like checking presence of fields
// Value checks, like expected length of string (to prevent timing attacks)
// Now, you could glue multiple services in the controller layer
// Or have another layer, if its too complicated
// But then more the layers, more the complexity of code
// Also, while SRP is great, there needs to be a discussion about
// Principle of locality.
// Its a balance between PoL and SRP. Tight coupling also has its place
// More layers, difficult it gets to retain things in head
// And more pointers and pc in cpu jumping from here and there and there
// That's atleast what I think as of 10th May,2023

type MetadataRequest struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	FileType string `json:"fileType"`
	Digest   string `json:"digest"`
	Chunks   int    `json:"chunks"`
}

type FileChunkRequest struct {
	ChunkID     string    `form:"id"`
	NextChunkID string    `form:"nextChunkID"`
	ChunkDigest string    `form:"chunkDigest"`
	ChunkSize   int       `form:"chunkSize"`
	Data        io.Reader `form:"-"`
	FileID      string    `form:"-"`
	UserID      string    `form:"-"`
}

type FileUpdateMetadataRequest struct {
	FileName     string `json:"fileName"`
	FileID       string `json:"fileID"`
	Digest       string `json:"digest"`
	Chunks       int64  `json:"chunks"`
	FileSize     int64  `json:"fileSize"`
	UploadStatus string `json:"-"`
	UserID       string `json:"-"`
}
