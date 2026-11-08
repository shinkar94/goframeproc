package resize

import (
	"github.com/shinkar94/goframeproc/pkg/constants"
)

type LookupTables struct {
	X0            []int    // Original X indices
    X1            []int    // X+1 indices for bilinear interpolation (with clamp)
    FxFixed       []uint32 // Fixed point for X interpolation
	Y0            []int    // Source Y indices
    Y1            []int    // Y+1 indices for bilinear interpolation (with clamp)
    FyFixed       []uint32 // Fixed point for Y interpolation
	SrcRowOffsets []int    // Row offsets in the source buffer
    DstRowOffsets []int    // Row offsets in the destination buffer

	XByteOffsets  []int // X0 * BytesPerPixel
	X1ByteOffsets []int // X1 * BytesPerPixel
}

// PrecomputeLookupTables calculates all coordinate tables once.
func PrecomputeLookupTables(srcW, srcH, targetW, targetH, stride int) *LookupTables {
	tables := &LookupTables{
		X0:            make([]int, targetW),
		X1:            make([]int, targetW),
		FxFixed:       make([]uint32, targetW),
		XByteOffsets:  make([]int, targetW),
		X1ByteOffsets: make([]int, targetW),
		Y0:            make([]int, targetH),
		Y1:            make([]int, targetH),
		FyFixed:       make([]uint32, targetH),
		SrcRowOffsets: make([]int, targetH),
		DstRowOffsets: make([]int, targetH),
	}

	// Precomputing X or Y mapping
	for x := 0; x < targetW; x++ {
		var xf float64
		if targetW > 1 {
			xf = float64(x) * float64(srcW-1) / float64(targetW-1)
		}
		ix := int(xf)
		if ix >= srcW {
			ix = srcW - 1
		}
		tables.X0[x] = ix
		tables.XByteOffsets[x] = ix * constants.BytesPerPixel

		x1 := ix + 1
		if x1 >= srcW {
			x1 = srcW - 1
		}
		tables.X1[x] = x1
		tables.X1ByteOffsets[x] = x1 * constants.BytesPerPixel

		frac := xf - float64(ix)
		if frac > 1.0 {
			frac = 1.0
		}
		tables.FxFixed[x] = uint32(frac * 65536.0)
	}

	for y := 0; y < targetH; y++ {
		var yf float64
		if targetH > 1 {
			yf = float64(y) * float64(srcH-1) / float64(targetH-1)
		}
		iy := int(yf)
		if iy >= srcH {
			iy = srcH - 1
		}
		tables.Y0[y] = iy

		y1 := iy + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		tables.Y1[y] = y1

		frac := yf - float64(iy)
		if frac > 1.0 {
			frac = 1.0
		}
		tables.FyFixed[y] = uint32(frac * 65536.0)
		tables.SrcRowOffsets[y] = iy * stride
		tables.DstRowOffsets[y] = y * targetW * constants.BytesPerPixel
	}

	return tables
}

func resizeRow(startY, endY int, params ResizeParams, tables *LookupTables) {
	isInputRGBA := params.InputFormat == constants.FormatRGBA
	isOutputRGBA := params.Format == constants.FormatRGBA

	src := params.Src
	dst := params.Dst
	srcLen := len(src)
	stride := params.Stride
	targetW := params.TargetW

	if params.Mode == Nearest {
		resizeNearest(startY, endY, src, dst, srcLen, targetW, tables, isInputRGBA, isOutputRGBA)
	} else {
		resizeBilinear(startY, endY, src, dst, srcLen, stride, targetW, tables, isInputRGBA, isOutputRGBA)
	}
}

func resizeNearest(startY, endY int, src, dst []byte, srcLen, targetW int, tables *LookupTables, isInputRGBA, isOutputRGBA bool) {
	maxIdx := srcLen - constants.BytesPerPixel
	if maxIdx < 0 {
		return // empty buffer
	}

	// Bounds check elimination hint for the Go compiler
	_ = src[maxIdx]

	sameFormat := isInputRGBA == isOutputRGBA

	for y := startY; y < endY; y++ {
		rowSrcOffset := tables.SrcRowOffsets[y]
		rowDstOffset := tables.DstRowOffsets[y]

		for x := 0; x < targetW; x++ {
			srcIdx := rowSrcOffset + tables.XByteOffsets[x]
			if srcIdx > maxIdx {
				srcIdx = maxIdx
			}

			dstIdx := rowDstOffset + x*constants.BytesPerPixel

			c0, c1, c2, c3 := src[srcIdx], src[srcIdx+1], src[srcIdx+2], src[srcIdx+3]

			if sameFormat {
				dst[dstIdx] = c0
				dst[dstIdx+1] = c1
				dst[dstIdx+2] = c2
				dst[dstIdx+3] = c3
			} else {
				dst[dstIdx] = c2
				dst[dstIdx+1] = c1
				dst[dstIdx+2] = c0
				dst[dstIdx+3] = c3
			}
		}
	}
}

// bilinear interpolation
func resizeBilinear(startY, endY int, src, dst []byte, srcLen, stride, targetW int, tables *LookupTables, isInputRGBA, isOutputRGBA bool) {
	maxIdx := srcLen - constants.BytesPerPixel
	if maxIdx < 0 {
		return // empty buffer
	}

	_ = src[maxIdx]

	for y := startY; y < endY; y++ {
		iy := tables.Y0[y]
		y1 := tables.Y1[y]
		fy := tables.FyFixed[y]
		rowDstOffset := tables.DstRowOffsets[y]

		row0Offset := iy * stride
		row1Offset := y1 * stride

		for x := 0; x < targetW; x++ {
			fx := tables.FxFixed[x]
			xOff := tables.XByteOffsets[x]
			x1Off := tables.X1ByteOffsets[x]

			i00 := row0Offset + xOff
			i10 := row0Offset + x1Off
			i01 := row1Offset + xOff
			i11 := row1Offset + x1Off

			// Clamp only if necessary (borders)
			if i11 > maxIdx {
				if i00 > maxIdx {
					i00 = maxIdx
				}
				if i10 > maxIdx {
					i10 = maxIdx
				}
				if i01 > maxIdx {
					i01 = maxIdx
				}
				i11 = maxIdx
			}

			var r00, g00, b00, a00 uint8
			var r10, g10, b10, a10 uint8
			var r01, g01, b01, a01 uint8
			var r11, g11, b11, a11 uint8

			if isInputRGBA {
				r00, g00, b00, a00 = src[i00], src[i00+1], src[i00+2], src[i00+3]
				r10, g10, b10, a10 = src[i10], src[i10+1], src[i10+2], src[i10+3]
				r01, g01, b01, a01 = src[i01], src[i01+1], src[i01+2], src[i01+3]
				r11, g11, b11, a11 = src[i11], src[i11+1], src[i11+2], src[i11+3]
			} else {
				b00, g00, r00, a00 = src[i00], src[i00+1], src[i00+2], src[i00+3]
				b10, g10, r10, a10 = src[i10], src[i10+1], src[i10+2], src[i10+3]
				b01, g01, r01, a01 = src[i01], src[i01+1], src[i01+2], src[i01+3]
				b11, g11, r11, a11 = src[i11], src[i11+1], src[i11+2], src[i11+3]
			}

			r := bilinearInterp(r00, r10, r01, r11, fx, fy)
			g := bilinearInterp(g00, g10, g01, g11, fx, fy)
			b := bilinearInterp(b00, b10, b01, b11, fx, fy)
			a := bilinearInterp(a00, a10, a01, a11, fx, fy)

			dstIdx := rowDstOffset + x*constants.BytesPerPixel
			if isOutputRGBA {
				dst[dstIdx] = r
				dst[dstIdx+1] = g
				dst[dstIdx+2] = b
				dst[dstIdx+3] = a
			} else {
				dst[dstIdx] = b
				dst[dstIdx+1] = g
				dst[dstIdx+2] = r
				dst[dstIdx+3] = a
			}
		}
	}
}

// fx, fy are guaranteed to be in the range [0, 65536] after PrecomputeLookupTables
func bilinearInterp(c00, c10, c01, c11 uint8, fx, fy uint32) uint8 {
	invFx := 65536 - fx
	invFy := 65536 - fy

	w00 := invFx * invFy
	w10 := fx * invFy
	w01 := invFx * fy
	w11 := fx * fy

	sum := uint64(c00)*uint64(w00) +
		uint64(c10)*uint64(w10) +
		uint64(c01)*uint64(w01) +
		uint64(c11)*uint64(w11)

	return uint8(sum >> 32)
}
