default:
	# Use static builds to make sure we don't have library version issues between the build env and lambda
	GOOS=linux GOARCH=amd64 go build -tags "osusergo netgo static_build lambda.norpc" -ldflags '-extldflags "-static"' -o bootstrap
	zip soci_index_generator_lambda.zip bootstrap

test:
	go test -v ./...
