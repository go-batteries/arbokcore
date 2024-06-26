--sql:CreateFileChunk

INSERT INTO user_files (
	user_id
	,file_id
	,chunk_id
	,next_chunk_id
	,chunk_blob_url
	,chunk_hash
	,created_at
	,updated_at
) VALUES (
	:user_id
	,:file_id
	,:chunk_id
	,:next_chunk_id
	,:chunk_blob_url
	,:chunk_hash
	,:created_at
	,:updated_at
);

--sql:GetChunkForFile
SELECT 
	user_id
	,chunk_id
	,next_chunk_id
	,chunk_blob_url
	,chunk_hash
	,created_at
	,updated_at
FROM user_files
WHERE file_id = :file_id
AND chunk_id = :chunk_id;


--sql:GetFileChunks

SELECT 
	user_id
	,chunk_id
	,next_chunk_id
	,chunk_blob_url
	,chunk_hash
	,created_at
	,updated_at
FROM user_files
WHERE file_id = :file_id
ORDER BY created_at  DESC;

--sql:GetFilesChunks

SELECT 
	user_id
	,chunk_id
	,next_chunk_id
	,chunk_blob_url
	,chunk_hash
	,created_at
	,updated_at
FROM user_files
WHERE file_id IN (:file_ids)
ORDER BY chunk_id DESC;
