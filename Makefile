run.setup:
	go run cmd/cli/main.go migrate -dir up

run.server:
	go run cmd/server/main.go

run.supervise.metadatas:
	go run cmd/workers/main.go supervise -name metadatas


build.testdata:
	mkdir -p tmp/testdata
	openssl rand -base64 $((100*1024*1024)) | head -c $((24*1024*1024)) > tmp/testdata/24mb.txt
	openssl rand -base64 $((100*1024*1024)) | head -c $((33*1024*1024)) > tmp/testdata/33mb.txt
	openssl rand -base64 $((100*1024*1024)) | head -c $((4*1024*1024)) > tmp/testdata/4mb.txt
