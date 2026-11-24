#include "textflag.h"

// func h21AVX2(dst []byte, src []byte, iters int)
// Each iteration processes 4 output pixels (16 bytes) via two 32-byte loads
// at +0 (px0..7) and +16 (px4..11). Every blend lives entirely in lane0 —
// no VPERMQ, no fix-up shuffle (same discipline as h32AVX2):
//
//	lane0(load1) = px0..3 -> [avg(p0,p1), avg(p2,p3)]
//	lane0(load2) = px4..7 -> [avg(p4,p5), avg(p6,p7)]
//	store        = lane0-interleave -> [avg01, avg23, avg45, avg67, ·, ·]
//
// Advances 32 source / 16 destination bytes per iteration; the second load
// reads up to 32*(iters-1)+47 — the caller keeps the row reads in bounds
// (ratio21_amd64.go). Registers: Y8 = maskE21, Y9 = maskO21.
TEXT ·h21AVX2(SB), NOSPLIT, $0-56
	MOVQ dst_base+0(FP), DI
	MOVQ src_base+24(FP), SI
	MOVQ iters+48(FP), CX
	TESTQ CX, CX
	JZ done
	VMOVDQU ·maskE21(SB), Y8
	VMOVDQU ·maskO21(SB), Y9
loop:
	VMOVDQU (SI), Y0       // lane0: px0..3
	VMOVDQU 16(SI), Y1     // lane0: px4..7
	VPSHUFB Y8, Y0, Y2     // lane0: [p0, p2]
	VPSHUFB Y9, Y0, Y3     // lane0: [p1, p3]
	VPAVGB  Y3, Y2, Y2     // lane0: [avg(p0,p1), avg(p2,p3)]
	VPSHUFB Y8, Y1, Y4     // lane0: [p4, p6]
	VPSHUFB Y9, Y1, Y5     // lane0: [p5, p7]
	VPAVGB  Y5, Y4, Y4     // lane0: [avg(p4,p5), avg(p6,p7)]
	VPUNPCKLQDQ Y4, Y2, Y2 // [avg01, avg23, avg45, avg67, ·, ·]
	VMOVDQU X2, (DI)
	ADDQ $32, SI
	ADDQ $16, DI
	DECQ CX
	JNZ loop
done:
	RET

// func hn21AVX2(dst []byte, src []byte, iters int)
// Nearest 2:1 horizontal: per iteration (two windows, 4 output pixels)
// pure pshufb copies of the even pixels [p0,p2] / [p4,p6] — no pavgb.
// Bounds identical to h21AVX2.
TEXT ·hn21AVX2(SB), NOSPLIT, $0-56
	MOVQ dst_base+0(FP), DI
	MOVQ src_base+24(FP), SI
	MOVQ iters+48(FP), CX
	TESTQ CX, CX
	JZ ndone
	VMOVDQU ·maskE21(SB), Y8
nloop:
	VMOVDQU (SI), Y0
	VMOVDQU 16(SI), Y1
	VPSHUFB Y8, Y0, Y2     // lane0: [p0, p2]
	VPSHUFB Y8, Y1, Y4     // lane0: [p4, p6]
	VPUNPCKLQDQ Y4, Y2, Y2 // [p0, p2, p4, p6, ·, ·]
	VMOVDQU X2, (DI)
	ADDQ $32, SI
	ADDQ $16, DI
	DECQ CX
	JNZ nloop
ndone:
	RET
