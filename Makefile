.PHONY: api vendor

api:
	protoc --go_out=. --go-grpc_out=. api/proto/v1/teleworker.proto

vendor:
	go mod tidy
	go mod vendor