#include "textflag.h"

// func SwapAVX2(src []byte, dst []byte)
TEXT ·SwapAVX2(SB), NOSPLIT, $0-48
	MOVQ src_base+0(FP), SI
	MOVQ src_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	CMPQ CX, $32
	JL done
	VMOVDQU ·maskAVX2Lo(SB), Y1
	VMOVDQU ·maskAVX2Hi(SB), Y2
	SHRQ $5, CX
loop:
	VMOVDQU (SI), Y0
	VPSHUFB Y1, Y0, Y3
	VPSHUFB Y2, Y0, Y4
	VPOR Y3, Y4, Y0
	VMOVDQU Y0, (DI)
	ADDQ $32, SI
	ADDQ $32, DI
	SUBQ $1, CX
	JNZ loop
done:
	RET

// func SwapSSE(src []byte, dst []byte)
TEXT ·SwapSSE(SB), NOSPLIT, $0-48
	MOVQ src_base+0(FP), SI
	MOVQ src_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	CMPQ CX, $16
	JL done
	MOVOU ·maskSSE(SB), X1
	SHRQ $4, CX
loop:
	MOVOU (SI), X0
	PSHUFB X1, X0
	MOVOU X0, (DI)
	ADDQ $16, SI
	ADDQ $16, DI
	SUBQ $1, CX
	JNZ loop
done:
	RET
