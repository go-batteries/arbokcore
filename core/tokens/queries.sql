--sql:CreateToken

INSERT INTO tokens (
	resource_id
	,resource_type
	,token_type
	,user_id
	,access_token
	,refresh_token
	,access_expires_at
	,refresh_expires_at
	,created_at
	,updated_at
) VALUES (
	:resource_id
	,:resource_type
	,:token_type
	,:user_id
	,:access_token
	,:refresh_token
	,:access_expires_at
	,:refresh_expires_at
	,:created_at
	,:updated_at
);

--sql:GetAccessToken

SELECT
	resource_id
	,resource_type
	,token_type
	,user_id
	,access_token
	,refresh_token
	,access_expires_at
	,refresh_expires_at
	,created_at
	,updated_at
FROM tokens
WHERE access_token = :access_token
AND resource_id = :resource_id
AND resource_type = :resource_type
LIMIT 1;

--sql:GetStreamToken

SELECT
	resource_id
	,resource_type
	,token_type
	,user_id
	,access_token
	,refresh_token
	,access_expires_at
	,refresh_expires_at
	,created_at
	,updated_at
FROM tokens
WHERE access_token = :access_token
AND resource_id = :resource_id
AND user_id = :user_id
AND resource_type = :resource_type
LIMIT 1;

