//go:build !amd64 && !386

package simd

// Non-x86 builds (arm, arm64, ...): no SSE features — scalar fallbacks.

func HaveSSSE3() bool { return false }
func HaveAVX2() bool  { return false }
