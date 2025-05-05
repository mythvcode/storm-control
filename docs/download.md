# Download Process

Storm-Control can be downloaded from the GitHub registry.

```bash
docker pull ghcr.io/mythvcode/storm-control:latest
```

For the program to function properly, the following capabilities are required: `BPF`, `NET_ADMIN`, `PERFMON`.

## Running as a specific user
```bash
docker run --rm   --cap-add=BPF   --cap-add=NET_ADMIN  --cap-add=PERFMON   --net=host --user user:group ghcr.io/mythvcode/storm-control:latest
```

## Running as root

The required capabilities must be explicitly specified:
```bash
docker run --rm   --cap-add=BPF   --cap-add=NET_ADMIN  --cap-add=PERFMON   --net=host --user 0 ghcr.io/mythvcode/storm-control:latest
```

## Using a Privileged Container
```bash
docker run --rm --privileged  --user 0 ghcr.io/mythvcode/storm-control:latest
```