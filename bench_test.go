package frameproc

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/shinkar94/goframeproc/pkg/constants"
	"github.com/shinkar94/goframeproc/pkg/resize"
)

func benchProc(b *testing.B, res string, mode resize.Mode, perf string) {
	src := testBuffer()
	opts := CompressionOptions{
		InputFormat:     "BGRA",
		OutputFormat:    "RGBA",
		Resolution:      constants.ResolutionPreset(res),
		Mode:            mode,
		PerformanceMode: perf,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ProcessFrame(src, testW, testH, testStride, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchProcInto(b *testing.B, res string, mode resize.Mode, perf string) {
	src := testBuffer()
	dst := make([]byte, testW*testH*4)
	opts := CompressionOptions{
		InputFormat:     "BGRA",
		OutputFormat:    "RGBA",
		Resolution:      constants.ResolutionPreset(res),
		Mode:            mode,
		PerformanceMode: perf,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ProcessFrameInto(dst, src, testW, testH, testStride, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNoResize_1080p(b *testing.B) { benchProc(b, "original", resize.Nearest, "saving") }
func BenchmarkInto_NoResize_1080p(b *testing.B) {
	benchProcInto(b, "original", resize.Nearest, "saving")
}

func Benchmark720p_Nearest_seq(b *testing.B)  { benchProc(b, "720p", resize.Nearest, "saving") }
func Benchmark720p_Nearest_par(b *testing.B)  { benchProc(b, "720p", resize.Nearest, "performance") }
func Benchmark720p_Bilinear_seq(b *testing.B) { benchProc(b, "720p", resize.Bilinear, "saving") }
func Benchmark720p_Bilinear_par(b *testing.B) { benchProc(b, "720p", resize.Bilinear, "performance") }

func BenchmarkInto_720p_Nearest_seq(b *testing.B) { benchProcInto(b, "720p", resize.Nearest, "saving") }
func BenchmarkInto_720p_Nearest_par(b *testing.B) {
	benchProcInto(b, "720p", resize.Nearest, "performance")
}
func BenchmarkInto_720p_Bilinear_seq(b *testing.B) {
	benchProcInto(b, "720p", resize.Bilinear, "saving")
}
func BenchmarkInto_720p_Bilinear_par(b *testing.B) {
	benchProcInto(b, "720p", resize.Bilinear, "performance")
}

func BenchmarkWeak2_720p_Nearest_seq(b *testing.B) {
	defer withGOMAXPROCS(b, 2)()
	benchProcInto(b, "720p", resize.Nearest, "saving")
}

func BenchmarkWeak2_720p_Bilinear_seq(b *testing.B) {
	defer withGOMAXPROCS(b, 2)()
	benchProcInto(b, "720p", resize.Bilinear, "saving")
}

func withGOMAXPROCS(b *testing.B, n int) func() {
	old := runtime.GOMAXPROCS(n)
	return func() { runtime.GOMAXPROCS(old) }
}

func init() { fmt.Printf("default GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0)) }
