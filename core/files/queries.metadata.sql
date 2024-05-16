--sql:CreateFileMetadata

INSERT INTO file_metadatas (
	id
	,prev_id
	,user_id
	,file_name
	,file_size
	,file_type
	,file_hash
	,chunks
	,current_flag
	,upload_status
	,created_at
	,updated_at
	,end_date
) VALUES (
	:id
	,:prev_id
	,:user_id
	,:file_name
	,:file_size
	,:file_type
	,:file_hash
	,:chunks
	,:current_flag
	,:upload_status
	,:created_at
	,:updated_at
	,:end_date
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
	,current_flag
	,created_at
FROM file_metadatas
WHERE
	user_id = :user_id
AND current_flag = 1
ORDER BY created_at DESC
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
	,fm.current_flag
	,fm.created_at
	,ufs.chunk_id
	,ufs.chunk_blob_url
	,ufs.chunk_hash
	,ufs.next_chunk_id
FROM file_metadatas fm
JOIN user_files ufs
ON
	ufs.file_id = fm.id
WHERE
	fm.user_id = :user_id
AND fm.current_flag = 1
ORDER BY fm.created_at DESC
LIMIT %d OFFSET %d;


--sql:UpdateCurrentFlag

UPDATE file_metadatas
SET
	current_flag = :current_flag
	,end_date = :end_date
WHERE
	id = :id
%s;


--sql:FindByHash

SELECT COUNT(1) as found 
FROM file_metadatas 
WHERE file_hash = ?;
