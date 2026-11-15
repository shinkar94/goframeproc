package frameproc

import (
	"encoding/binary"
	"errors"
	"image"
	"runtime"
	"sync"

	"github.com/shinkar94/goframeproc/pkg/constants"
	"github.com/shinkar94/goframeproc/pkg/simd"
	resize "github.com/shinkar94/goframeproc/pkg/resize"
)

// Validation errors
var (
	ErrEmptyBuffer     = errors.New("buffer is empty")
	ErrInvalidSize     = errors.New("invalid width or height")
	ErrBufferSizeMismatch = errors.New("buffer size does not match resolution")
)

type CompressionOptions struct {
	OutputFormat    string                     // constants.FormatBGRA or constants.FormatRGBA
	InputFormat     string                     // constants.FormatBGRA or constants.FormatRGBA
	Quality         int                        // 1-100 (reserved for future use)
	Resolution      constants.ResolutionPreset // "320p", "420p", "720p", "1080p", "original"
	Mode            resize.Mode
	PerformanceMode string // saving | performance
}


func BGRAtoImage(buffer []byte, width, height int) (*image.RGBA, error) {
	if err := validateInput(buffer, width, height); err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pix := img.Pix
	pixStride := img.Stride

	for y := 0; y < height; y++ {
		srcRowStart := y * width * 4
		dstRowStart := y * pixStride

		for x := 0; x < width; x++ {
			srcIdx := srcRowStart + x*4
			dstIdx := dstRowStart + x*4

			pix[dstIdx] = buffer[srcIdx+2]
			pix[dstIdx+1] = buffer[srcIdx+1]
			pix[dstIdx+2] = buffer[srcIdx]
			pix[dstIdx+3] = buffer[srcIdx+3]
		}
	}
	return img, nil
}

func RGBAtoImage(buffer []byte, width, height int) (*image.RGBA, error) {
	if err := validateInput(buffer, width, height); err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	if img.Stride == width*4 {
		copy(img.Pix, buffer)
	} else {
		for y := 0; y < height; y++ {
			srcStart := y * width * 4
			dstStart := y * img.Stride
			copy(img.Pix[dstStart:dstStart+width*4], buffer[srcStart:srcStart+width*4])
		}
	}
	return img, nil
}

func validateInput(buffer []byte, width, height int) error {
	if len(buffer) == 0 {
		return ErrEmptyBuffer
	}
	if width <= 0 || height <= 0 {
		return ErrInvalidSize
	}
	expectedSize := width * height * resize.BytesPerPixel
	if len(buffer) != expectedSize {
		return ErrBufferSizeMismatch
	}
	return nil
}

func resolveResolution(res constants.ResolutionPreset, w, h int) (int, int) {
	landscape := w >= h
	switch res {
	case constants.Res320p:
		if landscape {
			return 568, 320
		}
		return 320, 568
	case constants.Res420p:
		if landscape {
			return 740, 420
		}
		return 420, 740
	case constants.Res720p:
		if landscape {
			return 1280, 720
		}
		return 720, 1280
	case constants.Res1080p:
		if landscape {
			return 1920, 1080
		}
		return 1080, 1920
	default:
		return w, h
	}
}

func ResolveResolution(res constants.ResolutionPreset, w, h int) (int, int) {
	return resolveResolution(res, w, h)
}

func ProcessFrame(buffer []byte, width, height int, stride int, opts CompressionOptions) ([]byte, error) {
	return ProcessFrameInto(nil, buffer, width, height, stride, opts)
}

func ProcessFrameInto(dst []byte, buffer []byte, width, height int, stride int, opts CompressionOptions) ([]byte, error) {
	if len(buffer) == 0 {
		return nil, ErrEmptyBuffer
	}
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidSize
	}

	targetW, targetH := resolveResolution(opts.Resolution, width, height)
	needed := targetW * targetH * resize.BytesPerPixel

	if cap(dst) < needed {
		dst = make([]byte, needed)
	}
	dst = dst[:needed]

	if targetW == width && targetH == height {
		if opts.InputFormat == opts.OutputFormat {
			copy(dst, buffer)
			return dst, nil
		}
		convertFormatInto(dst, buffer, width*height, opts.PerformanceMode == "performance")
		return dst, nil
	}

	var resizeStrategy resize.ResizeStrategy
	if opts.PerformanceMode == "performance" {
		resizeStrategy = resize.ParallelResizeStrategy{}
	} else {
		resizeStrategy = resize.SequentialResizeStrategy{}
	}

	return resize.ResizeGenericInto(dst, buffer, width, height, stride, targetW, targetH, opts.Mode, opts.OutputFormat, resizeStrategy, opts.InputFormat), nil
}

func convertFormatInto(dst, src []byte, pixelCount int, parallel bool) {
	if pixelCount >= 8 {
		bytes := pixelCount * 4
		if simd.HaveAVX2() {
			full := bytes &^ 31
			simd.SwapAVX2(src[:full], dst[:full])
			tailSwap(dst[full:], src[full:])
			return
		}
		if simd.HaveSSSE3() {
			full := bytes &^ 15
			simd.SwapSSE(src[:full], dst[:full])
			tailSwap(dst[full:], src[full:])
			return
		}
	}

	swap := func(start, end int) {
		for i := start; i < end; i++ {
			v := binary.LittleEndian.Uint32(src[i*4 : i*4+4])
			binary.LittleEndian.PutUint32(dst[i*4:i*4+4], (v&0xFF00FF00)|((v&0x00FF0000)>>16)|((v&0x000000FF)<<16))
		}
	}

	if !parallel || pixelCount < 480000 {
		swap(0, pixelCount)
		return
	}

	workers := runtime.GOMAXPROCS(0)
	if workers > 8 {
		workers = 8
	}
	if workers > 1 {
		chunk := (pixelCount + workers - 1) / workers
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			start := w * chunk
			if start >= pixelCount {
				break
			}
			end := start + chunk
			if end > pixelCount {
				end = pixelCount
			}
			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()
				swap(start, end)
			}(start, end)
		}
		wg.Wait()
		return
	}
	swap(0, pixelCount)
}

func tailSwap(dst, src []byte) {
	for i := 0; i+4 <= len(src); i += 4 {
		v := binary.LittleEndian.Uint32(src[i : i+4])
		binary.LittleEndian.PutUint32(dst[i:i+4], (v&0xFF00FF00)|((v&0x00FF0000)>>16)|((v&0x000000FF)<<16))
	}
}
