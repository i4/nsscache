all:
	go generate
	go vet
	go build
	$(MAKE) --no-print-directory -C nss all

clean:
	rm -f nsscash filetype_string.go
	$(MAKE) --no-print-directory -C nss clean

test:
	go build # we need ./nsscash
	go test
	$(MAKE) --no-print-directory -C nss test

.PHONY: all clean test
