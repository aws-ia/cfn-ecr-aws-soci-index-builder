default:
	# Use static builds to make sure we don't have library version issues between the build env and lambda
	GOOS=linux GOARCH=amd64 go build -tags "osusergo netgo static_build" -ldflags '-extldflags "-static"' -o main
	zip soci_index_generator_lambda.zip main

test:
	go test -v ./...
