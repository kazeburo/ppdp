VERSION=0.0.2
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

all: ppdp

.PHONY: ppdp

bundle:
	dep ensure

update:
	dep ensure -update

ppdp: ppdp.go
	go build $(LDFLAGS) ppdp.go

linux: ppdp.go
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
