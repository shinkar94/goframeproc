#include "textflag.h"

// func h21SSE386(dst []byte, src []byte, iters int)
// XMM twin of h21AVX2: each iteration processes 4 output pixels (16 bytes)
// via two 16-byte loads at +0 (px0..3) and +16 (px4..7). Per lane (whole
// register is one lane on XMM): [p0,p2]/[p1,p3] and [p4,p6]/[p5,p7] pshufb
// extraction, PAVGB pairs, PUNPCKLQDQ interleave:
//
//	X2 = pavgb([p0,p2], [p1,p3]) -> [avg01, avg23]
//	X4 = pavgb([p4,p6], [p5,p7]) -> [avg45, avg67]
//	store = [avg01, avg23, avg45, avg67]
//
// Advances exactly 32 source / 16 destination bytes per iteration; loads
// never overshoot the row, so the caller may run all dstPx/4 iterations.
// Registers: X4 = maskE21_386, X5 = maskO21_386. Slice headers on 386 are
// 12 bytes, iters adds 4 — frame $0-28.
TEXT ·h21SSE386(SB), NOSPLIT, $0-28
	MOVL dst_base+0(FP), DI
	MOVL src_base+12(FP), SI
	MOVL iters+24(FP), CX
	TESTL CX, CX
	JZ done
	MOVOU ·maskE21_386(SB), X4
	MOVOU ·maskO21_386(SB), X5
loop:
	MOVOU (SI), X0       // px0..3
	MOVOU 16(SI), X1     // px4..7
	MOVOU X0, X2
	MOVOU X1, X3
	PSHUFB X4, X0        // [p0, p2]
	PSHUFB X5, X2        // [p1, p3]
	PSHUFB X4, X1        // [p4, p6]
	PSHUFB X5, X3        // [p5, p7]
	PAVGB X2, X0         // [avg(p0,p1), avg(p2,p3)]
	PAVGB X3, X1         // [avg(p4,p5), avg(p6,p7)]
	PUNPCKLQDQ X1, X0    // [avg01, avg23, avg45, avg67]
	MOVOU X0, (DI)
	ADDL $32, SI
	ADDL $16, DI
	DECL CX
	JNZ loop
done:
	RET

// func hn21SSE386(dst []byte, src []byte, iters int)
// Nearest 2:1 horizontal: per iteration (two windows, 4 output pixels)
// pure pshufb copies of the even pixels [p0,p2] / [p4,p6] — no pavgb.
// Bounds identical to h21SSE386.
TEXT ·hn21SSE386(SB), NOSPLIT, $0-28
	MOVL dst_base+0(FP), DI
	MOVL src_base+12(FP), SI
	MOVL iters+24(FP), CX
	TESTL CX, CX
	JZ ndone
	MOVOU ·maskE21_386(SB), X4
nloop:
	MOVOU (SI), X0
	MOVOU 16(SI), X1
	PSHUFB X4, X0        // [p0, p2]
	PSHUFB X4, X1        // [p4, p6]
	PUNPCKLQDQ X1, X0    // [p0, p2, p4, p6]
	MOVOU X0, (DI)
	ADDL $32, SI
	ADDL $16, DI
	DECL CX
	JNZ nloop
ndone:
	RET

// func avgSSE386(dst []byte, a []byte, b []byte)
// dst[i] = (a[i] + b[i] + 1) >> 1 over floor(len(a)/64) 64-byte quads,
// unrolled ×4 for ILP (four independent load->pavgb->store chains).
// The tail is handled by the caller. Frame: three 12-byte slices = $0-36.
TEXT ·avgSSE386(SB), NOSPLIT, $0-36
	MOVL dst_base+0(FP), DI
	MOVL a_base+12(FP), SI
	MOVL b_base+24(FP), DX
	MOVL a_len+16(FP), CX
	CMPL CX, $64
	JL done
	SHRL $6, CX
loop:
	MOVOU (SI), X0
	MOVOU (DX), X1
	MOVOU 16(SI), X2
	MOVOU 16(DX), X3
	MOVOU 32(SI), X4
	MOVOU 32(DX), X5
	MOVOU 48(SI), X6
	MOVOU 48(DX), X7
	PAVGB X1, X0
	PAVGB X3, X2
	PAVGB X5, X4
	PAVGB X7, X6
	MOVOU X0, (DI)
	MOVOU X2, 16(DI)
	MOVOU X4, 32(DI)
	MOVOU X6, 48(DI)
	ADDL $64, SI
	ADDL $64, DX
	ADDL $64, DI
	DECL CX
	JNZ loop
done:
	RET
