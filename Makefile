all:
	go generate ./...
	go vet ./...
	go build
	go test ./...

clean:
	rm -f nsscash filetype_string.go

.PHONY: all clean
