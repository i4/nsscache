all:
	go generate ./...
	go vet ./...
	go build
	go test ./...
	$(MAKE) --no-print-directory -C nss all

clean:
	rm -f nsscash filetype_string.go
	$(MAKE) --no-print-directory -C nss clean

test:
	go build # we need ./nsscash
	$(MAKE) --no-print-directory -C nss test

.PHONY: all clean test
