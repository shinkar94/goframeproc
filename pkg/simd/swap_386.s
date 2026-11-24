#include "textflag.h"

// func swapSSE386(src []byte, dst []byte)
// 386 twin of the amd64 swapSSE: processes floor(len(src)/16)*16 bytes with
// SSSE3 PSHUFB over XMM. Slice headers on 386 are 12 bytes each, hence the
// $0-24 frame and the +12 offset for dst.
TEXT ·swapSSE386(SB), NOSPLIT, $0-24
	MOVL src_base+0(FP), SI
	MOVL src_len+4(FP), CX
	MOVL dst_base+12(FP), DI
	CMPL CX, $16
	JL done
	MOVOU ·maskSSE386(SB), X1
	SHRL $4, CX
loop:
	MOVOU (SI), X0
	PSHUFB X1, X0
	MOVOU X0, (DI)
	ADDL $16, SI
	ADDL $16, DI
	SUBL $1, CX
	JNZ loop
done:
	RET
