//go:build !amd64

package simd

func HaveSSSE3() bool { return false }
func HaveAVX2() bool  { return false }
