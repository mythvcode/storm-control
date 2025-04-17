package ebpfxdp

import (
	_ "embed"
)

//go:embed kernel/xdp_kernel.o
var KernelProgramBytes []byte
