--sql:CreateUserQuery

INSERT INTO users(
	id
	,user_type
	,email
	,hashedpass
	,created_at
	,updated_at
) VALUES (
	:id
	,:user_type
	,:email
	,:hashedpass
	,:created_at
	,:updated_at
);


--sql:GetUserByEmail

SELECT 
	id
	,user_type
	,email
	,hashedpass
FROM users
WHERE 
	email = :email AND 
	user_type = :user_type;

