#include "textflag.h"

// func h32AVX2(dst []byte, src []byte, iters int)
// Each iteration processes 4 groups (8 output pixels, 32 bytes) via two
// 32-byte loads at +0 and +12 (windows px0..7 and px3..10 for source
// px0..11). Every blend lives entirely in lane0 — no VPERMQ, no fix-up
// shuffle:
//
//	out 4g   = px0  (window1 lane0 dword0, used directly)
//	out 4g+1 = pavgb(px1, px2)
//	out 4g+2 = px3  (window2 lane0 dword0)
//	out 4g+3 = pavgb(px4, px5)
//
// Two windows use disjoint registers so the OoO engine overlaps their
// latency chains. Advances 48 source / 32 destination bytes per iteration.
// Reads up to 48*(iters-1)+68 source bytes — the caller keeps the final
// groups scalar so every load stays in row bounds (ratio32_amd64.go).
//
// Registers: Y8 = maskP [4..11], Y9 = maskT [8..15];
// window1: Y0,L0 | Y1,L1 | Y2 | Y3 | Y4 | Y5; window2: Y6,Y7,Y10,Y11,Y12,Y13.
TEXT ·h32AVX2(SB), NOSPLIT, $0-56
	MOVQ dst_base+0(FP), DI
	MOVQ src_base+24(FP), SI
	MOVQ iters+48(FP), CX
	TESTQ CX, CX
	JZ done
	VMOVDQU ·hMaskP(SB), Y8
	VMOVDQU ·hMaskT(SB), Y9
loop:
	VMOVDQU (SI), Y0
	VMOVDQU 12(SI), Y1
	VMOVDQU 24(SI), Y6
	VMOVDQU 36(SI), Y7
	// window 1: groups (px0,1,2) and (px3,4,5)
	VPSHUFB Y8, Y0, Y2 // [p1, p2]
	VPSHUFB Y9, Y0, Y3 // [p2, p3]
	VPAVGB  Y3, Y2, Y2 // a12
	VPSHUFB Y8, Y1, Y4 // [p4, p5]
	VPSHUFB Y9, Y1, Y5 // [p5, p6]
	VPAVGB  Y5, Y4, Y4 // a45
	VPUNPCKLDQ Y2, Y0, Y2 // [p0, a12, ·, ·]
	VPUNPCKLDQ Y4, Y1, Y4 // [p3, a45, ·, ·]
	VPUNPCKLQDQ Y4, Y2, Y2 // [p0, a12, p3, a45]
	VMOVDQU X2, (DI)
	// window 2: groups (px6,7,8) and (px9,10,11)
	VPSHUFB Y8, Y6, Y10
	VPSHUFB Y9, Y6, Y11
	VPAVGB  Y11, Y10, Y10
	VPSHUFB Y8, Y7, Y12
	VPSHUFB Y9, Y7, Y13
	VPAVGB  Y13, Y12, Y12
	VPUNPCKLDQ Y10, Y6, Y10
	VPUNPCKLDQ Y12, Y7, Y12
	VPUNPCKLQDQ Y12, Y10, Y10
	VMOVDQU X10, 16(DI)
	ADDQ $48, SI
	ADDQ $32, DI
	DECQ CX
	JNZ loop
done:
	RET

// func avgAVX2(dst []byte, a []byte, b []byte)
// dst[i] = (a[i] + b[i] + 1) >> 1 over floor(len(a)/128) 128-byte quads,
// unrolled ×4 for ILP (four independent load->pavgb->store chains).
// The tail is handled by the caller.
TEXT ·avgAVX2(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DI
	MOVQ a_base+24(FP), SI
	MOVQ b_base+48(FP), DX
	MOVQ a_len+32(FP), CX
	CMPQ CX, $128
	JL done
	SHRQ $7, CX
loop:
	VMOVDQU (SI), Y0
	VMOVDQU (DX), Y8
	VMOVDQU 32(SI), Y1
	VMOVDQU 32(DX), Y9
	VMOVDQU 64(SI), Y2
	VMOVDQU 64(DX), Y10
	VMOVDQU 96(SI), Y3
	VMOVDQU 96(DX), Y11
	VPAVGB Y8, Y0, Y0
	VPAVGB Y9, Y1, Y1
	VPAVGB Y10, Y2, Y2
	VPAVGB Y11, Y3, Y3
	VMOVDQU Y0, (DI)
	VMOVDQU Y1, 32(DI)
	VMOVDQU Y2, 64(DI)
	VMOVDQU Y3, 96(DI)
	ADDQ $128, SI
	ADDQ $128, DX
	ADDQ $128, DI
	DECQ CX
	JNZ loop
done:
	RET

// func hn32AVX2(dst []byte, src []byte, iters int)
// Nearest 3:2 horizontal: per iteration (two windows, 8 output pixels):
//   out 4g   = px 6g,   out 4g+1 = px 6g+1   (first 8 bytes of window1)
//   out 4g+2 = px 6g+3, out 4g+3 = px 6g+4   (first 8 bytes of window2)
// Pure pshufb copies — no pavgb. Advances 48 source / 32 destination bytes;
// bounds identical to h32AVX2 (caller keeps the tail scalar).
TEXT ·hn32AVX2(SB), NOSPLIT, $0-56
	MOVQ dst_base+0(FP), DI
	MOVQ src_base+24(FP), SI
	MOVQ iters+48(FP), CX
	TESTQ CX, CX
	JZ ndone
	VMOVDQU ·hMaskPair(SB), Y8
nloop:
	VMOVDQU (SI), Y0
	VMOVDQU 12(SI), Y1
	VMOVDQU 24(SI), Y5
	VMOVDQU 36(SI), Y6
	VPSHUFB Y8, Y0, Y2      // [p0, p1]
	VPSHUFB Y8, Y1, Y3      // [p3, p4]
	VPUNPCKLQDQ Y3, Y2, Y2  // [p0, p1, p3, p4]
	VMOVDQU X2, (DI)
	VPSHUFB Y8, Y5, Y7      // [p6, p7]
	VPSHUFB Y8, Y6, Y9      // [p9, p10]
	VPUNPCKLQDQ Y9, Y7, Y7  // [p6, p7, p9, p10]
	VMOVDQU X7, 16(DI)
	ADDQ $48, SI
	ADDQ $32, DI
	DECQ CX
	JNZ nloop
ndone:
	RET
