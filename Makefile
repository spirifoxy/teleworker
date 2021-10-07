.PHONY: api vendor

api:
	protoc \
		--proto_path=api/proto \
		--go_out=. \
		--go-grpc_out=. \
		--validate_out="lang=go:." \
		api/proto/v1/teleworker.proto

vendor:
	go mod tidy
	go mod vendor