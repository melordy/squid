RELEASE=release

BINARY=squid

all: run

run:
	go run app/*.go

build:
	CGO_ENABLED=0 go build -o ${BINARY} app/*.go

release:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${RELEASE}/${BINARY}_linux_amd64 app/*.go
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o ${RELEASE}/${BINARY}_windows_amd64 app/*.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o ${RELEASE}/${BINARY}_darwin_amd64 app/*.go

test:
	go test -cover -covermode=atomic ./...

clean:
	rm -rf ${BINARY} ${RELEASE}

docker:
	docker build -t artifactory.nike.com:9001/quality/iqe/ocv2:${VERSION} .