package resize

// Ratio32 — fixed-ratio 3:2 bilinear downscale with libyuv-style specialized
// kernels. At an exact 3:2 ratio the bilinear phases cycle with period 2:
//
//	out x = 2k   -> src pixel 3k            (pure copy, weight 1.0)
//	out x = 2k+1 -> src pixels 3k+1, 3k+2   (50/50 blend, pavgb rounding)
//
// The same cycle holds vertically for rows. That removes the per-pixel table
// gather entirely: source is consumed contiguously (24 bytes per 2 output
// pixels) with pshufb extraction and VPAVGB blends.
//
// Mapping note: this is the exact-ratio mapping src = 3/2 * dst, while the
// generic path (BilinearBGRA) maps edge-to-edge x*(srcW-1)/(dstW-1). For
// 1920x1080 -> 1280x720 the positions drift by up to 0.5 px near the right
// edge; the difference is quantified in TestRatio32VsEdgeToEdgeReference.
// goframeproc semantics are NOT changed here — lab only.
//
// Requirements: dstW, dstH even; srcW == 3*dstW/2; srcH == 3*dstH/2;
// stride >= srcW*4. Otherwise NewRatio32 returns nil and callers should fall
// back to the generic path.

type Ratio32 struct {
	srcW, srcH, stride int
	dstW, dstH         int
	groups             int // dstW/2 horizontal 2-output units
	pairs              int // dstH/2 vertical 2-row units
	inter              []byte
}

func NewRatio32(srcW, srcH, stride, dstW, dstH int) *Ratio32 {
	if dstW <= 0 || dstH <= 0 || dstW%2 != 0 || dstH%2 != 0 {
		return nil
	}
	if srcW != 3*dstW/2 || srcH != 3*dstH/2 {
		return nil
	}
	if stride < srcW*4 {
		return nil
	}
	return &Ratio32{
		srcW: srcW, srcH: srcH, stride: stride,
		dstW: dstW, dstH: dstH,
		groups: dstW / 2, pairs: dstH / 2,
		inter: make([]byte, dstW*srcH*4),
	}
}

// Into downscales src into dst reusing the preallocated intermediate —
// zero allocations per frame after NewRatio32.
func (r *Ratio32) Into(dst, src []byte) {
	w4 := r.dstW * 4
	if len(dst) < r.dstW*r.dstH*4 {
		panic("ratio32: dst too small")
	}
	if len(src) < (r.srcH-1)*r.stride+r.srcW*4 {
		panic("ratio32: src too small")
	}
	// Horizontal: srcW x srcH -> dstW x srcH.
	for y := 0; y < r.srcH; y++ {
		dispatchH32(r.inter[y*w4:(y+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.groups)
	}
	// Vertical: dstW x srcH -> dstW x dstH.
	// Row 2k is a pure copy of row 3k; row 2k+1 is pavgb(rows 3k+1, 3k+2).
	for k := 0; k < r.pairs; k++ {
		copy(dst[(2*k)*w4:(2*k+1)*w4], r.inter[(3*k)*w4:(3*k+1)*w4])
		dispatchAvg(dst[(2*k+1)*w4:(2*k+2)*w4],
			r.inter[(3*k+1)*w4:(3*k+2)*w4],
			r.inter[(3*k+2)*w4:(3*k+3)*w4])
	}
}

// IntoNearest is the nearest-neighbour variant of the 3:2 downscale.
// Exact-ratio floor mapping (same rounding as goframeproc nearest, which
// truncates int(xf)): out x=2k <- px 3k, out x=2k+1 <- px 3k+1 — pure copies,
// no blends; vertical rows are copies too, and every third source row is
// never sampled, so the horizontal pass skips it entirely.
func (r *Ratio32) IntoNearest(dst, src []byte) {
	w4 := r.dstW * 4
	if len(dst) < r.dstW*r.dstH*4 {
		panic("ratio32: dst too small")
	}
	if len(src) < (r.srcH-1)*r.stride+r.srcW*4 {
		panic("ratio32: src too small")
	}
	for y := 0; y < r.srcH; y++ {
		if y%3 == 2 {
			continue // row y=3k+2 is never selected by floor(1.5*out_y)
		}
		dispatchH32Nearest(r.inter[y*w4:(y+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.groups)
	}
	for k := 0; k < r.pairs; k++ {
		copy(dst[(2*k)*w4:(2*k+1)*w4], r.inter[(3*k)*w4:(3*k+1)*w4])
		copy(dst[(2*k+1)*w4:(2*k+2)*w4], r.inter[(3*k+1)*w4:(3*k+2)*w4])
	}
}

// ratio32NearestScalarPath is the all-scalar path of IntoNearest (no SIMD
// dispatch), byte-identical to the SIMD result — used by tests to pin the asm.
func ratio32NearestScalarPath(dst, src []byte, r *Ratio32) {
	w4 := r.dstW * 4
	for y := 0; y < r.srcH; y++ {
		if y%3 == 2 {
			continue
		}
		h32NearestScalar(r.inter[y*w4:(y+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.groups)
	}
	for k := 0; k < r.pairs; k++ {
		copy(dst[(2*k)*w4:(2*k+1)*w4], r.inter[(3*k)*w4:(3*k+1)*w4])
		copy(dst[(2*k+1)*w4:(2*k+2)*w4], r.inter[(3*k+1)*w4:(3*k+2)*w4])
	}
}

// h32NearestScalar writes `groups` horizontal units (8 bytes each): unit g
// copies source pixels 3g and 3g+1 (8 contiguous bytes), pixel 3g+2 skipped.
func h32NearestScalar(dstRow, srcRow []byte, groups int) {
	for g := 0; g < groups; g++ {
		copy(dstRow[g*8:(g+1)*8], srcRow[g*12:g*12+8])
	}
}

// h32Scalar writes `groups` horizontal units (8 bytes each): unit g copies
// src pixel 3g and pavgb-blends pixels 3g+1, 3g+2.
func h32Scalar(dstRow, srcRow []byte, groups int) {
	for g := 0; g < groups; g++ {
		s, o := g*12, g*8
		copy(dstRow[o:o+4], srcRow[s:s+4])
		avg4(dstRow[o+4:o+8], srcRow[s+4:s+8], srcRow[s+8:s+12])
	}
}

func avg4(dst, a, b []byte) {
	for i := 0; i < 4; i++ {
		dst[i] = byte((uint16(a[i]) + uint16(b[i]) + 1) >> 1)
	}
}

// avgScalar is the portable row averager with pavgb semantics.
func avgScalar(dst, a, b []byte) {
	for i := range dst {
		dst[i] = byte((uint16(a[i]) + uint16(b[i]) + 1) >> 1)
	}
}

// ratio32TwoPassScalar is the all-scalar path of Into (no SIMD dispatch),
// byte-identical to the SIMD result — used by tests to pin the asm.
func ratio32TwoPassScalar(dst, src []byte, r *Ratio32) {
	w4 := r.dstW * 4
	for y := 0; y < r.srcH; y++ {
		h32Scalar(r.inter[y*w4:(y+1)*w4], src[y*r.stride:y*r.stride+r.srcW*4], r.groups)
	}
	for k := 0; k < r.pairs; k++ {
		copy(dst[(2*k)*w4:(2*k+1)*w4], r.inter[(3*k)*w4:(3*k+1)*w4])
		avgScalar(dst[(2*k+1)*w4:(2*k+2)*w4],
			r.inter[(3*k+1)*w4:(3*k+2)*w4],
			r.inter[(3*k+2)*w4:(3*k+3)*w4])
	}
}

// Ratio32Ref4Tap is an independent single-pass reference with 32-bit weights
// and the exact-ratio mapping (fx, fy in {0, 32768}). Same arithmetic family
// as ScalarBilinear4Tap; used for tolerance checks (expected max diff 2 LSB:
// one per 16-bit pass).
func Ratio32Ref4Tap(dst, src []byte, srcW, srcH, stride, dstW, dstH int) {
	for y := 0; y < dstH; y++ {
		iy := (3 * y) / 2
		fy := uint32(3*y&1) * 32768
		y1 := iy + 1
		row0, row1 := src[iy*stride:], src[y1*stride:]
		for x := 0; x < dstW; x++ {
			ix := (3 * x) / 2
			fx := uint32(3*x&1) * 32768
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
