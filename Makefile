.PHONY: api vendor build test certgen

OUT_DIR=./out
SEC_DIR=./security

api:
	protoc \
		--proto_path=api/proto \
		--go_out=. \
		--go-grpc_out=. \
		api/proto/v1/teleworker.proto
vendor:
	go mod tidy
	go mod vendor

build:
	go build -o ${OUT_DIR}/teleworker ./client
	go build -o ${OUT_DIR}/twserver ./server

test:
	go vet ./...
	go test -v ./... -race

certgen:
	openssl genpkey -algorithm ed25519 > ${SEC_DIR}/ca-key.pem
	openssl req -new -x509 -config ${SEC_DIR}/openssl.conf \
   		-key ${SEC_DIR}/ca-key.pem \
   		-out ${SEC_DIR}/ca-cert.pem

	openssl genpkey -algorithm ed25519 > ${SEC_DIR}/server-key.pem
	openssl req -new -config ${SEC_DIR}/openssl.conf \
		-key ${SEC_DIR}/server-key.pem \
		-out ${SEC_DIR}/server-req.pem
	openssl x509 -req -CAcreateserial -extensions req_ext \
		-extfile ${SEC_DIR}/openssl.conf \
		-in ${SEC_DIR}/server-req.pem \
		-CA ${SEC_DIR}/ca-cert.pem \
		-CAkey ${SEC_DIR}/ca-key.pem \
		-out ${SEC_DIR}/server-cert.pem

	openssl genpkey -algorithm ed25519 > ${SEC_DIR}/client-key.pem
	openssl req -new -subj /C=CZ/L=Prague/CN=client \
		-key ${SEC_DIR}/client-key.pem \
		-out ${SEC_DIR}/client-req.pem
	openssl x509 -req -CAcreateserial \
		-in ${SEC_DIR}/client-req.pem \
		-CA ${SEC_DIR}/ca-cert.pem \
		-CAkey ${SEC_DIR}/ca-key.pem \
		-out ${SEC_DIR}/client-cert.pem
