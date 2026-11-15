#include "textflag.h"

// func cpuid(leaf, subleaf uint32) (eax, ebx, ecx, edx uint32)
TEXT ·cpuid(SB), NOSPLIT, $0-24
	MOVL leaf+0(FP), AX
	MOVL subleaf+4(FP), CX
	CPUID
	MOVL AX, eax+8(FP)
	MOVL BX, ebx+12(FP)
	MOVL CX, ecx+16(FP)
	MOVL DX, edx+20(FP)
	RET

// func xgetbv0() (low, high uint32)
TEXT ·xgetbv0(SB), NOSPLIT, $0-8
	XORL CX, CX
	XGETBV
	MOVL AX, low+0(FP)
	MOVL DX, high+4(FP)
	RET
