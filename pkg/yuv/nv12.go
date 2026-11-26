// Package yuv converts BGRA frames to NV12: desktop capture gives BGRA,
// H.264 encoders consume NV12.
//
// Semantics: BT.601 video-range with rounded shifts; chroma is a rounded
// 2:2 box average computed before the color formula:
//
//	y  = ((66r + 129g + 25b + 128) >> 8) + 16
//	mc = (c00 + c01 + c10 + c11 + 2) >> 2
//	u  = ((-38mr - 74mg + 112mb + 128) >> 8) + 128
//	v  = ((112mr - 94mg - 18mb + 128) >> 8) + 128
//
// With s32 arithmetic the results stay in [16..235] / [16..240], so no
// clamping is needed. The scalar implementation is the spec; the SIMD path
// matches it byte-for-byte (pinned by tests).
//
// Requirements: w/h even and positive, stride >= 4*w. dst is packed NV12:
// h rows of w bytes (Y), then h/2 rows of w bytes (interleaved UV).
package yuv

// NV12 is a reusable conversion plan: geometry precomputed, zero
// allocations per frame after construction.
type NV12 struct {
	w, h, stride int
}

// NewNV12 validates the geometry and returns a plan, or nil if invalid
// (callers fall back to a generic path).
func NewNV12(w, h, stride int) *NV12 {
	if w <= 0 || h <= 0 || w%2 != 0 || h%2 != 0 || stride < w*4 {
		return nil
	}
	return &NV12{w: w, h: h, stride: stride}
}

// Size returns the required dst length: Y plane + UV plane.
func (p *NV12) Size() int { return p.w * (p.h + p.h/2) }

// Into converts one BGRA frame into dst, reusing the plan — zero
// allocations per frame.
func (p *NV12) Into(dst, src []byte) {
	if len(dst) < p.Size() {
		panic("yuv: dst too small")
	}
	if len(src) < (p.h-1)*p.stride+p.w*4 {
		panic("yuv: src too small")
	}
	w := p.w
	for y := 0; y < p.h; y++ {
		dispatchY(dst[y*w:(y+1)*w], src[y*p.stride:y*p.stride+w*4], w)
	}
	for cy := 0; cy < p.h/2; cy++ {
		dispatchUV(dst[(p.h+cy)*w:(p.h+cy+1)*w],
			src[(2*cy)*p.stride:(2*cy)*p.stride+w*4],
			src[(2*cy+1)*p.stride:(2*cy+1)*p.stride+w*4],
			w/2)
	}
}
