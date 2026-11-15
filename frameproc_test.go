package frameproc

import (
	"bytes"
	"testing"

	"github.com/shinkar94/goframeproc/pkg/constants"
	"github.com/shinkar94/goframeproc/pkg/resize"
)

const (
	testW, testH = 1920, 1080
	testStride   = testW * 4
)

func testBuffer() []byte {
	b := make([]byte, testW*testH*4)
	for i := 0; i < len(b); i += 4 {
		b[i] = byte(i % 256)
		b[i+1] = byte((i / 7) % 256)
		b[i+2] = byte((i / 13) % 256)
		b[i+3] = 255
	}
	return b
}

func testOptions(res string, mode resize.Mode, perf string) CompressionOptions {
	return CompressionOptions{
		InputFormat:     "BGRA",
		OutputFormat:    "BGRA",
		Resolution:      constants.ResolutionPreset(res),
		Mode:            mode,
		PerformanceMode: perf,
	}
}

func TestBilinearCornersNotZero(t *testing.T) {
	src := testBuffer()
	out, err := ProcessFrame(src, testW, testH, testStride, testOptions("720p", resize.Bilinear, "saving"))
	if err != nil {
		t.Fatal(err)
	}
	tw, th := 1280, 720
	corners := [][2]int{{0, 0}, {tw - 1, 0}, {0, th - 1}, {tw - 1, th - 1}}
	for _, p := range corners {
		idx := (p[1]*tw + p[0]) * 4
		if out[idx+3] == 0 {
			t.Errorf("corner (%d,%d): alpha=0", p[0], p[1])
		}
	}
}

func TestConvertFormatIntoMatchesByteSwap(t *testing.T) {
	src := testBuffer()
	want := make([]byte, len(src))
	for i := 0; i < len(src); i += 4 {
		want[i], want[i+1], want[i+2], want[i+3] = src[i+2], src[i+1], src[i], src[i+3]
	}
	for _, parallel := range []bool{false, true} {
		got := make([]byte, len(src))
		convertFormatInto(got, src, testW*testH, parallel)
		if !bytes.Equal(got, want) {
			t.Fatalf("convertFormatInto(parallel=%v) mismatch", parallel)
		}
	}
}

func TestProcessFrameMatchesInto(t *testing.T) {
	src := testBuffer()
	for _, res := range []string{"original", "720p", "420p"} {
		for _, mode := range []resize.Mode{resize.Nearest, resize.Bilinear} {
			opts := testOptions(res, mode, "saving")
			a, errA := ProcessFrame(src, testW, testH, testStride, opts)
			b, errB := ProcessFrameInto(nil, src, testW, testH, testStride, opts)
			if errA != nil || errB != nil {
				t.Fatalf("ProcessFrame/Into error: %v / %v", errA, errB)
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("res=%s mode=%v: ProcessFrame differs from ProcessFrameInto", res, mode)
			}
		}
	}
}

func TestProcessFrameIntoReusesBuffer(t *testing.T) {
	src := testBuffer()
	dst := make([]byte, testW*testH*4)
	opts := testOptions("original", resize.Nearest, "saving")

	allocs := testing.AllocsPerRun(10, func() {
		out, err := ProcessFrameInto(dst, src, testW, testH, testStride, opts)
		if err != nil {
			t.Fatal(err)
		}
		if &out[0] != &dst[0] {
			t.Fatal("dst buffer was not reused")
		}
	})
	if allocs != 0 {
		t.Errorf("ProcessFrameInto with reused dst allocated %v times, want 0", allocs)
	}
}
