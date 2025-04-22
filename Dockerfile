ARG GO_VERSION

# Build ebpf program
FROM --platform=amd64 ubuntu:24.04 as ebpfbuilder

WORKDIR /build
RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections
RUN apt update && apt install clang libbpf-dev make -y
ADD . ./
RUN make build_xdp

# Build userspace ebpf program
FROM golang:${GO_VERSION} as gobuilder

WORKDIR /build
ADD . ./
COPY --from=ebpfbuilder /build/ebpfxdp/kernel/xdp_kernel.o ./ebpfxdp/kernel/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o storm-control ./cmd/stormcontrol

# Copy builded programs to alpine image
FROM alpine:latest

LABEL description="Ebpf based storm control and broadcast and multicast exporter program."
RUN addgroup --gid 39555 storm_control && \
    adduser -h /app -s /bin/sh -G storm_control -u 39555 -D storm_control
WORKDIR /app/
COPY --from=gobuilder /build/storm-control .

USER storm_control

ENTRYPOINT ["/app/storm-control"]


