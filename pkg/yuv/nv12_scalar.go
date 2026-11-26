package yuv

// yScalar is the per-row Y reference: one Y byte per BGRA pixel.
func yScalar(dstY, srcRow []byte, w int) {
	for x := 0; x < w; x++ {
		i := x * 4
		b := int(srcRow[i])
		g := int(srcRow[i+1])
		r := int(srcRow[i+2])
		dstY[x] = byte((66*r+129*g+25*b+128)>>8 + 16)
	}
}

// uvScalar is the per-row chroma reference: one rounded 2x2 box average
// per channel, then the color formula. dstUV holds u,v pairs; w2 = w/2.
func uvScalar(dstUV, srcRow0, srcRow1 []byte, w2 int) {
	for x := 0; x < w2; x++ {
		i0 := x * 8 // first pixel of the pair
		i1 := i0 + 4
		mb := (int(srcRow0[i0]) + int(srcRow0[i1]) + int(srcRow1[i0]) + int(srcRow1[i1]) + 2) >> 2
		mg := (int(srcRow0[i0+1]) + int(srcRow0[i1+1]) + int(srcRow1[i0+1]) + int(srcRow1[i1+1]) + 2) >> 2
		mr := (int(srcRow0[i0+2]) + int(srcRow0[i1+2]) + int(srcRow1[i0+2]) + int(srcRow1[i1+2]) + 2) >> 2
		u := ((-38*mr - 74*mg + 112*mb + 128) >> 8) + 128
		v := ((112*mr - 94*mg - 18*mb + 128) >> 8) + 128
		dstUV[2*x] = byte(u)
		dstUV[2*x+1] = byte(v)
	}
}

// ScalarBGRAtoNV12 is the portable reference implementation (no dispatch).
func ScalarBGRAtoNV12(dst, src []byte, w, h, stride int) {
	p := NewNV12(w, h, stride)
	if p == nil {
		panic("yuv: invalid geometry")
	}
	if len(dst) < p.Size() {
		panic("yuv: dst too small")
	}
	if len(src) < (h-1)*stride+w*4 {
		panic("yuv: src too small")
	}
	for y := 0; y < h; y++ {
		yScalar(dst[y*w:(y+1)*w], src[y*stride:y*stride+w*4], w)
	}
	for cy := 0; cy < h/2; cy++ {
		uvScalar(dst[(h+cy)*w:(h+cy+1)*w],
			src[(2*cy)*stride:(2*cy)*stride+w*4],
			src[(2*cy+1)*stride:(2*cy+1)*stride+w*4],
			w/2)
	}
}
