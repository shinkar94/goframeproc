package resize

import "github.com/shinkar94/goframeproc/pkg/constants"

// Box (area-average) resampling: each output pixel is the average of the
// source rectangle it covers. At downscale it is the aliasing-free filter —
// unlike bilinear, which decimates and can skip thin lines entirely
// (subpixel-rendered text loses its ClearType fringing too). At upscale the
// source rect degenerates to a single pixel and Box degrades to
// nearest-neighbour behavior; use Bilinear for upscaling.
//
// The implementation is direct rect summation (no lookup tables needed —
// rect bounds are derived from the output coordinates), so it works with
// the parallel strategy unchanged: every output row is independent.

func resizeBox(startY, endY int, src, dst []byte, srcW, srcH, stride, targetW, targetH int, isInputRGBA, isOutputRGBA bool) {
	srcLen := len(src)
	if srcLen == 0 || targetW <= 0 || targetH <= 0 {
		return
	}
	_ = src[srcLen-1] // bounds check elimination hint

	sameFormat := isInputRGBA == isOutputRGBA

	for y := startY; y < endY; y++ {
		sy0, sy1 := boxRect(y, srcH, targetH)
		rowDstOffset := y * targetW * constants.BytesPerPixel

		for x := 0; x < targetW; x++ {
			sx0, sx1 := boxRect(x, srcW, targetW)

			var c [4]uint32
			var cnt uint32
			for yy := sy0; yy < sy1; yy++ {
				row := src[yy*stride:]
				for xx := sx0; xx < sx1; xx++ {
					p := row[xx*constants.BytesPerPixel : xx*constants.BytesPerPixel+4]
					c[0] += uint32(p[0])
					c[1] += uint32(p[1])
					c[2] += uint32(p[2])
					c[3] += uint32(p[3])
					cnt++
				}
			}
			if cnt == 0 {
				continue
			}
			// round-half-up per channel
			v := [4]byte{
				uint8((c[0] + cnt/2) / cnt),
				uint8((c[1] + cnt/2) / cnt),
				uint8((c[2] + cnt/2) / cnt),
				uint8((c[3] + cnt/2) / cnt),
			}

			dstIdx := rowDstOffset + x*constants.BytesPerPixel
			if sameFormat {
				dst[dstIdx], dst[dstIdx+1], dst[dstIdx+2], dst[dstIdx+3] = v[0], v[1], v[2], v[3]
			} else {
				dst[dstIdx], dst[dstIdx+1], dst[dstIdx+2], dst[dstIdx+3] = v[2], v[1], v[0], v[3]
			}
		}
	}
}

// boxRect returns the half-open source range [start, end) covered by output
// index i of n outputs over srcSize. Always at least one pixel (upscale /
// degenerate case: nearest-like); clamped to the source bounds.
func boxRect(i, srcSize, n int) (int, int) {
	start := i * srcSize / n
	end := (i + 1) * srcSize / n
	if end <= start {
		end = start + 1
	}
	if end > srcSize {
		end = srcSize
	}
	return start, end
}
