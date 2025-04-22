LIBC_HEADERS=/usr/include/x86_64-linux-gnu
GOLANG_CI_VERSION ?= 'v1.59.1'

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o storm-control ./cmd/stormcontrol

build_xdp:
	clang  -target bpf -I ${LIBC_HEADERS} -g -O2 -o ./ebpfxdp/kernel/xdp_kernel.o -c ebpfxdp/kernel/xdp_kernel.c

tests:
	go test -v ./...

lint:
	golangci-lint run -v ./...


create_test_files:
	@if [ ! -f ebpfxdp/kernel/xdp_kernel.o ]; then\
	    echo "create ebpfxdp/kernel/xdp_kernel.o";\
		touch ebpfxdp/kernel/xdp_kernel.o;\
	else\
		echo "file ebpfxdp/kernel/xdp_kernel.o aready exist";\
	fi

clean:
	@if [ ! -s ebpfxdp/kernel/xdp_kernel.o ]; then\
		echo "rm -rf ebpfxdp/kernel/xdp_kernel.o";\
		rm -rf ebpfxdp/kernel/xdp_kernel.o;\
	else\
		echo "file ebpfxdp/kernel/xdp_kernel.o not empty, skip deletion";\
	fi
	rm -rf ./storm-control

install_linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $(GOLANG_CI_VERSION)
