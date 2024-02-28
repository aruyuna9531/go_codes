rm -rf ./proto_codes
mkdir ./proto_codes
protoc --go_out=. ./proto/*.proto