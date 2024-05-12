--sql:CreateFileMetadata

INSERT INTO file_metadatas (
	id
	,user_id
	,file_name
	,file_size
	,file_type
	,file_hash
	,chunks
	,upload_status
	,created_at
	,updated_at
) VALUES (
	:id
	,:user_id
	,:file_name
	,:file_size
	,:file_type
	,:file_hash
	,:chunks
	,:upload_status
	,:created_at
	,:updated_at
);

--sql:GetMetadataForUser

SELECT 
	id
	,user_id
	,file_name
	,file_size
	,file_type
	,file_hash
	,chunks
	,created_at
FROM file_metadatas
WHERE
	user_id = :user_id
LIMIT %d OFFSET %d;


--sql:GetFilesForUser

SELECT 
	fm.id
	,fm.user_id
	,fm.file_name
	,fm.file_size
	,fm.file_type
	,fm.file_hash
	,fm.chunks
	,fm.created_at
	,ufs.chunk_id
	,ufs.chunk_blob_url
	,ufs.chunk_hash
	,ufs.next_chunk_id
	,ufs.version
FROM file_metadatas fm
JOIN user_files ufs
ON
	ufs.file_id = fm.id
WHERE
	fm.user_id = :user_id
LIMIT %d OFFSET %d;

--sql:UpdateFileMetadata

UPDATE file_metadatas
SET
	file_hash = :file_hash
	,file_size = :file_size
	,upload_status = :upload_status
	,chunks = :chunks
	,updated_at = :updated_at
WHERE
	id = :id
AND user_id = :user_id;
