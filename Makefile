LIBC_HEADERS=/usr/include/x86_64-linux-gnu
GOLANG_CI_VERSION ?= 'v1.59.1'

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o storm-control ./cmd/stormcontrol

build_xdp:
	clang  -target bpf -I ${LIBC_HEADERS} -g -O2 -o ./ebpfxdp/kernel/xdp_kernel.o -c ebpfxdp/kernel/xdp_kernel.c

tests:
	go test -v ./...

lint:
	golangci-lint run -v ./...

install_linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $(GOLANG_CI_VERSION)
