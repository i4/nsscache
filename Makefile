all:
	go generate ./...
	go vet ./...
	go build
	go test ./...

clean:
	rm -f nsscash filetype_string.go

test:
	go build # we need ./nsscash
	$(MAKE) --no-print-directory -C nss test

.PHONY: all clean test
