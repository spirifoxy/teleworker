.PHONY: api

api:
	protoc --go_out=. --go-grpc_out=. api/proto/v1/teleworker.proto