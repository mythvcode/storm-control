LIBC_HEADERS=/usr/include/x86_64-linux-gnu

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o storm-control ./cmd/stormcontrol

build_xdp:
	clang  -target bpf -I ${LIBC_HEADERS} -g -O2 -o ./ebpfxdp/xdp_kernel.o -c ebpfxdp/kernel/xdp_kernel.c

tests:
	go test -v ./...

lint:
	golangci-lint run -v ./...
