.PHONY: api vendor

api:
	protoc \
		--proto_path=api/proto \
		--go_out=. \
		--go-grpc_out=. \
		api/proto/v1/teleworker.proto
vendor:
	go mod tidy
	go mod vendor