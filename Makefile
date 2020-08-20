VERSION=0.1.0
LDFLAGS=-ldflags "-X main.Version=${VERSION}"
GO111MODULE=on

all: ppdp

.PHONY: ppdp

ppdp: ppdp.go dumper/*.go upstream/*.go
	go build $(LDFLAGS) ppdp.go

linux: ppdp.go dumper/*.go upstream/*.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) ppdp.go

check:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -rf ppdp ppdp-*.tar.gz

tag:
	git tag v${VERSION}
	git push origin v${VERSION}
	git push origin master
	goreleaser --rm-dist
