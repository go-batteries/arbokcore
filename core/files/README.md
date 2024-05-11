## file service


Comprises of:

- file metadata service
- file upload download and cdc


### File metadata service

- Takes the user info and file info and saves them to a db.
- In response it returns a short lived access token and streamID
- File upload in chunks uses this to validate and associate a file upload

- This is also responsible for catalogging of what data belongs to user
- Manage Delete of files
- This is a fairly read heavy service




### File upload service

- receives a stream token, stream id, access_token
- validates the stream authorization
- uploads the chunks to either `BlobStore`
- `BlobStore` is an interface, it can be `S3`, `LocalFS`, `Ipfs` etc

- Also handles scatter gather, to get
    - All file chunks
    - All changed chunks

- Should also handle update file chunk


### User Files

This is a join table which is responsible for access control and sharing of files.

- Changes to files should be propagated to files shared by user




### File upload worker

- Workers should be able to get the chunks, combine them, validate the hash, check for virus etc. (For laters)
