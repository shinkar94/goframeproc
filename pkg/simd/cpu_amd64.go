//go:build amd64

package simd

func cpuid(leaf, subleaf uint32) (eax, ebx, ecx, edx uint32)
func xgetbv0() (low, high uint32)

// CPU feature detection is cached at package init: CPUID is a serializing
// instruction (~500 ns for the three leaves + XGETBV used here), and the
// dispatchers query it per frame/row — re-querying per call would cost
// ~1 ms per 1080p frame.
var (
	haveSSSE3 = detectSSSE3()
	haveAVX2  = detectAVX2()
	haveSSE41 = detectSSE41()
)

func detectSSSE3() bool {
	_, _, ecx, _ := cpuid(1, 0)
	return ecx&(1<<9) != 0
}

// detectSSE41 reports SSE4.1 (PMOVZXBW widening) — needed by pkg/yuv in
// addition to SSSE3's pshufb.
func detectSSE41() bool {
	_, _, ecx, _ := cpuid(1, 0)
	return ecx&(1<<19) != 0
}

// detectAVX2 reports AVX2 support with OS YMM state enabled (XGETBV).
func detectAVX2() bool {
	_, _, ecx, _ := cpuid(1, 0)
	if ecx&(1<<27) == 0 { // OSXSAVE
		return false
	}
	low, _ := xgetbv0()
	if low&6 != 6 { // XMM and YMM state must be enabled by OS
		return false
	}
	if maxLeaf, _, _, _ := cpuid(0, 0); maxLeaf < 7 {
		return false
	}
	_, ebx, _, _ := cpuid(7, 0)
	return ebx&(1<<5) != 0
}

// HaveSSSE3 reports SSSE3 support (pshufb).
func HaveSSSE3() bool { return haveSSSE3 }

// HaveSSE41 reports SSE4.1 support (PMOVZXBW widening).
func HaveSSE41() bool { return haveSSE41 }

// HaveAVX2 reports AVX2 support with OS YMM state enabled.
func HaveAVX2() bool { return haveAVX2 }
