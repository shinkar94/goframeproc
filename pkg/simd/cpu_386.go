//go:build 386

package simd

import "golang.org/x/sys/cpu"

// HaveSSSE3 reports SSSE3 support (pshufb). Real CPU feature detection via
// golang.org/x/sys/cpu: the previous stub always reported false, while
// 386-class CPUs (e.g. Silvermont Atoms) do support SSSE3/SSE4.2 — the stub
// made SSE paths permanently unavailable on windows/386. amd64 keeps its
// own asm-based detection (cpu_amd64.go).
func HaveSSSE3() bool { return cpu.X86.HasSSSE3 }

// HaveSSE41 reports SSE4.1 (PMOVZXBW widening, pkg/yuv); Silvermont-class
// tablets have SSE4.2 and pass.
func HaveSSE41() bool { return cpu.X86.HasSSE41 }

// HaveAVX2 reports AVX2 support; x/sys/cpu already includes the OSXSAVE +
// XCR0 YMM-state check in the HasAVX2 flag.
func HaveAVX2() bool { return cpu.X86.HasAVX2 }
