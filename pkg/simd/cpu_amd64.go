//go:build amd64

package simd

func cpuid(leaf, subleaf uint32) (eax, ebx, ecx, edx uint32)
func xgetbv0() (low, high uint32)

func HaveSSSE3() bool {
	_, _, ecx, _ := cpuid(1, 0)
	return ecx&(1<<9) != 0
}

func HaveAVX2() bool {
	_, _, ecx, _ := cpuid(1, 0)
	if ecx&(1<<27) == 0 {
		return false
	}
	low, _ := xgetbv0()
	if low&6 != 6 {
		return false
	}
	if maxLeaf, _, _, _ := cpuid(0, 0); maxLeaf < 7 {
		return false
	}
	_, ebx, _, _ := cpuid(7, 0)
	return ebx&(1<<5) != 0
}
