package resize

import (
	"github.com/shinkar94/goframeproc/pkg/constants"
	"runtime"
	"sync"
)

type Mode int

const (
	Nearest Mode = iota
	Bilinear
)

const BytesPerPixel = constants.BytesPerPixel

var tablesCache sync.Map // tablesCacheKey -> *LookupTables

type tablesCacheKey struct {
	srcW, srcH, targetW, targetH, stride int
}

func lookupTablesCached(srcW, srcH, targetW, targetH, stride int) *LookupTables {
	key := tablesCacheKey{srcW, srcH, targetW, targetH, stride}
	if t, ok := tablesCache.Load(key); ok {
		return t.(*LookupTables)
	}
	t := PrecomputeLookupTables(srcW, srcH, targetW, targetH, stride)
	tablesCache.Store(key, t)
	return t
}

func ResizeGeneric(src []byte, srcW, srcH, stride, targetW, targetH int, mode Mode, format string, resizeStrategy ResizeStrategy, inputFormat string) []byte {
	dst := make([]byte, targetW*targetH*constants.BytesPerPixel)
	return ResizeGenericInto(dst, src, srcW, srcH, stride, targetW, targetH, mode, format, resizeStrategy, inputFormat)
}

func ResizeGenericInto(dst []byte, src []byte, srcW, srcH, stride, targetW, targetH int, mode Mode, format string, resizeStrategy ResizeStrategy, inputFormat string) []byte {
	dst = dst[:targetW*targetH*constants.BytesPerPixel]

	tables := lookupTablesCached(srcW, srcH, targetW, targetH, stride)

	params := ResizeParams{
		Src:         src,
		SrcW:        srcW,
		SrcH:        srcH,
		Stride:      stride,
		TargetW:     targetW,
		TargetH:     targetH,
		Mode:        mode,
		Format:      format,
		InputFormat: inputFormat,
		Dst:         dst,
	}

	resizeStrategy.DoResize(params, tables)
	return dst
}

func (SequentialResizeStrategy) DoResize(params ResizeParams, tables *LookupTables) {
	resizeRow(0, params.TargetH, params, tables)
}

func (ParallelResizeStrategy) DoResize(params ResizeParams, tables *LookupTables) {
	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > params.TargetH {
		numWorkers = params.TargetH
	}

	var wg sync.WaitGroup
	rowsPerWorker := params.TargetH / numWorkers

	for w := 0; w < numWorkers; w++ {
		startY := w * rowsPerWorker
		endY := startY + rowsPerWorker
		if w == numWorkers-1 {
			endY = params.TargetH
		}
		wg.Add(1)
		go func(startY, endY int) {
			defer wg.Done()
			resizeRow(startY, endY, params, tables)
		}(startY, endY)
	}
	wg.Wait()
}
