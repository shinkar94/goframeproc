package resize

// Ratio21 — fixed-ratio 2:1 downscale with pavgb averaging, specialized for
// 16:10 sources (the tablet panel: 1280x800 -> 640x400).
//
// At an exact 2:1 the bilinear phases would collapse to pure copies under the
// ratio32 "sample at ratio*x" mapping (2k is always integer), so the 2:1
// kernel uses the area-centered mapping instead: each output pixel is the
// 50/50 blend of its 2x2 source block. That is simultaneously what bilinear
// with weights {0.5, 0.5} computes and what a 2x2 box would average —
// implemented as two pavgb passes (horizontal then vertical):
//
//	out x = k -> pavgb(src 2k, src 2k+1)
//	out y = k -> pavgb(row 2k, row 2k+1)
//
// No gather: source is consumed contiguously (8 bytes in per 4 bytes out).
// Two-pass rounding differs from the single-pass 4-tap reference by up to
// 2 LSB — checked in tests against Ratio21Ref4Tap (same tolerance family as
// ratio32).
//
// Mapping note vs the generic path: ResizeGeneric maps edge-to-edge
// x*(srcW-1)/(dstW-1); positions drift by up to ~1 px near the edges — the
// same drift family as ratio32, quantified in its tests.
//
// Requirements: srcW == 2*dstW, srcH == 2*dstH, stride >= srcW*4. Otherwise
// NewRatio21 returns nil and callers should fall back to the generic path.

type Ratio21 struct {
	srcW, srcH, stride int
	dstW, dstH         int
	inter              []byte
}

func NewRatio21(srcW, srcH, stride, dstW, dstH int) *Ratio21 {
	if dstW <= 0 || dstH <= 0 {
		return nil
	}
	if srcW != 2*dstW || srcH != 2*dstH {
		return nil
	}
	if stride < srcW*4 {
		return nil
	}
	return &Ratio21{
		srcW: srcW, srcH: srcH, stride: stride,
		dstW: dstW, dstH: dstH,
		inter: make([]byte, dstW*srcH*4),
	}
}

// Into downscales src into dst reusing the preallocated intermediate —
// zero allocations per frame after NewRatio21.
func (r *Ratio21) Into(dst, src []byte) {
	w4 := r.dstW * 4
	if len(dst) < r.dstW*r.dstH*4 {
		panic("ratio21: dst too small")
	}
	if len(src) < (r.srcH-1)*r.stride+r.srcW*4 {
		panic("ratio21: src too small")
	}
	// Horizontal: srcW x srcH -> dstW x srcH.
	for y := 0; y < r.srcH; y++ {
		dispatchH21(r.inter[y*w4:(y+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.dstW)
	}
	// Vertical: dstW x srcH -> dstW x dstH; out row k = pavgb(rows 2k, 2k+1).
	for k := 0; k < r.dstH; k++ {
		dispatchAvg21(dst[k*w4:(k+1)*w4],
			r.inter[(2*k)*w4:(2*k+1)*w4],
			r.inter[(2*k+1)*w4:(2*k+2)*w4])
	}
}

// IntoNearest is the nearest-neighbour variant of the 2:1 downscale.
// Floor mapping (same rounding as the package nearest, which truncates
// int(xf)): out px k <- src px 2k, out row k <- src row 2k — pure copies,
// no blends; odd source rows are never sampled, so the horizontal pass
// skips them entirely.
func (r *Ratio21) IntoNearest(dst, src []byte) {
	w4 := r.dstW * 4
	if len(dst) < r.dstW*r.dstH*4 {
		panic("ratio21: dst too small")
	}
	if len(src) < (r.srcH-1)*r.stride+r.srcW*4 {
		panic("ratio21: src too small")
	}
	for y := 0; y < r.srcH; y += 2 {
		dispatchH21Nearest(r.inter[(y/2)*w4:(y/2+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.dstW)
	}
	// inter holds exactly dstH rows (one per even source row): out row k
	// copies inter row k.
	for k := 0; k < r.dstH; k++ {
		copy(dst[k*w4:(k+1)*w4], r.inter[k*w4:(k+1)*w4])
	}
}

// h21Scalar writes dstPx output pixels (4 bytes each): px k blends source
// pixels 2k and 2k+1 with pavgb rounding.
func h21Scalar(dstRow, srcRow []byte, dstPx int) {
	for k := 0; k < dstPx; k++ {
		avg4(dstRow[k*4:k*4+4], srcRow[k*8:k*8+4], srcRow[k*8+4:k*8+8])
	}
}

// h21NearestScalar writes dstPx output pixels: px k copies source pixel 2k.
func h21NearestScalar(dstRow, srcRow []byte, dstPx int) {
	for k := 0; k < dstPx; k++ {
		copy(dstRow[k*4:k*4+4], srcRow[k*8:k*8+4])
	}
}

// ratio21TwoPassScalar is the all-scalar path of Into (no SIMD dispatch),
// byte-identical to the SIMD result — used by tests to pin the asm.
func ratio21TwoPassScalar(dst, src []byte, r *Ratio21) {
	w4 := r.dstW * 4
	for y := 0; y < r.srcH; y++ {
		h21Scalar(r.inter[y*w4:(y+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.dstW)
	}
	for k := 0; k < r.dstH; k++ {
		avgScalar(dst[k*w4:(k+1)*w4],
			r.inter[(2*k)*w4:(2*k+1)*w4],
			r.inter[(2*k+1)*w4:(2*k+2)*w4])
	}
}

// ratio21NearestScalarPath is the all-scalar path of IntoNearest (no SIMD
// dispatch), byte-identical to the SIMD result — used by tests to pin the asm.
func ratio21NearestScalarPath(dst, src []byte, r *Ratio21) {
	w4 := r.dstW * 4
	for y := 0; y < r.srcH; y += 2 {
		h21NearestScalar(r.inter[(y/2)*w4:(y/2+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.dstW)
	}
	for k := 0; k < r.dstH; k++ {
		copy(dst[k*w4:(k+1)*w4], r.inter[k*w4:(k+1)*w4])
	}
}

// Ratio21Ref4Tap is an independent single-pass reference with 32-bit weights
// and the area-centered mapping (fx, fy = 32768 constant = 0.5 in Q16.16).
// Same arithmetic family as Ratio32Ref4Tap; used for tolerance checks
// (expected max diff 2 LSB: one per 16-bit pass).
func Ratio21Ref4Tap(dst, src []byte, srcW, srcH, stride, dstW, dstH int) {
	const fx, fy = 32768, 32768
	for y := 0; y < dstH; y++ {
		iy := 2 * y
		row0, row1 := src[iy*stride:], src[(iy+1)*stride:]
		for x := 0; x < dstW; x++ {
			ix := 2 * x
			x1 := ix + 1
			i00 := row0[ix*4 : ix*4+4]
			i10 := row0[x1*4 : x1*4+4]
			i01 := row1[ix*4 : ix*4+4]
			i11 := row1[x1*4 : x1*4+4]
			wTop := uint64(65536-fx) * uint64(65536-fy)
			wTop1 := uint64(fx) * uint64(65536-fy)
			wBot := uint64(65536-fx) * uint64(fy)
			wBot1 := uint64(fx) * uint64(fy)
			for c := 0; c < 4; c++ {
				sum := uint64(i00[c])*wTop + uint64(i10[c])*wTop1 +
					uint64(i01[c])*wBot + uint64(i11[c])*wBot1
				dst[(y*dstW+x)*4+c] = uint8(sum >> 32)
			}
		}
	}
}
