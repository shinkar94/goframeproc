//go:build 386

package simd

import (
	"testing"

	"golang.org/x/sys/cpu"
)

// The wrapper must mirror the x/sys/cpu source of truth exactly.
func TestDetectionMatchesXSysCPU(t *testing.T) {
	if HaveSSSE3() != cpu.X86.HasSSSE3 {
		t.Fatalf("HaveSSSE3 = %v, want %v", HaveSSSE3(), cpu.X86.HasSSSE3)
	}
	if HaveAVX2() != cpu.X86.HasAVX2 {
		t.Fatalf("HaveAVX2 = %v, want %v", HaveAVX2(), cpu.X86.HasAVX2)
	}
	t.Logf("CPU flags: SSSE3=%v AVX2=%v (this machine)", cpu.X86.HasSSSE3, cpu.X86.HasAVX2)
}
