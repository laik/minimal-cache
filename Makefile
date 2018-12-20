all: gen build
PWD?=$(shell pwd)

gen:
	# @docker run --rm -v ${PWD}:${PWD} -w ${PWD} znly/protoc --gogofast_out=plugins=grpc:. --swagger_out=logtostderr=true:.  -I. *.proto
	protoc \
		-I api \
		-I $$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/ \
		-I $$GOPATH/src/github.com/gogo/googleapis/ \
		-I $$GOPATH/src/ \
		--gogo_out=plugins=grpc,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types:\
$$GOPATH/src/ \
		--grpc-gateway_out=\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types:\
$$GOPATH/src/ \
		--swagger_out=third_party/OpenAPI/ \
		--govalidators_out=gogoimport=true,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types:\
$$GOPATH/src \
api/*.proto

	# Workaround for https://github.com/grpc-ecosystem/grpc-gateway/issues/229.
	sed -i.bak "s/empty.Empty/types.Empty/g" api/api.pb.gw.go && rm api/api.pb.gw.go.bak

	# Generate static assets for OpenAPI UI
	statik -m -f -src third_party/OpenAPI/

build:
	# Building 
	go build -o bin/minimal-cache-server cmd/server/main.go
	go build -o bin/minimal-cache-cli cmd/client/main.go
