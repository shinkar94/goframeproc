#include "textflag.h"

// func yRowSSE(dstY []byte, srcRow []byte)
// 386 twin of the amd64 kernel: 12-byte slice headers, $0-24 frame,
// X0-X7 only (SSE adds no registers before AVX), constants via memory.
// SSE4.1 (Silvermont tablets pass; SSSE3-only Merom falls back to scalar).
TEXT ·yRowSSE(SB), NOSPLIT, $0-24
	MOVL dstY_base+0(FP), DI
	MOVL dstY_len+4(FP), CX
	MOVL srcRow_base+12(FP), SI
	SHRL $2, CX
	JZ ydone
yloop:
	MOVOU (SI), X0
	MOVO X0, X1
	PSHUFB ·yMaskRG(SB), X1
	MOVO X1, X2
	PMOVZXBW X2, X1
	PMADDWL ·yMadRG(SB), X1 // 66r + 129g per pixel
	MOVO X0, X2
	PSHUFB ·yMaskB0(SB), X2
	MOVO X2, X0
	PMOVZXBW X0, X2
	PMADDWL ·yMadB(SB), X2 // 25b
	PADDL X2, X1
	PADDL ·dY4224(SB), X1
	PSRAL $8, X1
	MOVO X1, X0
	PACKSSLW X0, X1
	MOVO X1, X0
	PACKUSWB X0, X1
	MOVD X1, (DI)
	ADDL $16, SI
	ADDL $4, DI
	DECL CX
	JNZ yloop
ydone:
	RET

// func uvRowSSE(dstUV []byte, srcRow0 []byte, srcRow1 []byte)
// 386 twin of the amd64 UV kernel. $0-36 frame: three 12-byte slices,
// X0-X7 only.
TEXT ·uvRowSSE(SB), NOSPLIT, $0-36
	MOVL dstUV_base+0(FP), DI
	MOVL dstUV_len+4(FP), CX
	MOVL srcRow0_base+12(FP), SI
	MOVL srcRow1_base+24(FP), DX
	SHRL $2, CX
	JZ uvdone
uvloop:
	MOVOU (SI), X0
	MOVOU (DX), X1
	MOVO X0, X2
	PSHUFB ·uvPair(SB), X2
	PMADDUBSW ·uvOnes(SB), X2 // row0: words [sumB, sumG, sumR, 0, ...]
	MOVO X1, X3
	PSHUFB ·uvPair(SB), X3
	PMADDUBSW ·uvOnes(SB), X3 // row1
	PADDSW X3, X2             // 2x2 sums
	PADDW ·uvTwo(SB), X2
	PSRLW $2, X2              // means [mb0,mg0,mr0,0, mb1,mg1,mr1,0]
	MOVO X2, X3
	PSHUFB ·uvRG(SB), X3      // (mr,mg) pairs
	MOVO X2, X4
	PSHUFB ·uvMB(SB), X4      // (mb,0) pairs
	// U
	MOVO X3, X5
	PMADDWL ·mU1(SB), X5      // -38mr -74mg
	MOVO X4, X6
	PMADDWL ·mU2(SB), X6      // 112mb
	PADDL X6, X5
	PADDL ·dUV32896(SB), X5
	PSRAL $8, X5              // [u0, u1, 0, 0]
	// V
	MOVO X3, X6
	PMADDWL ·mV1(SB), X6      // 112mr -94mg
	MOVO X4, X7
	PMADDWL ·mV2(SB), X7      // -18mb
	PADDL X7, X6
	PADDL ·dUV32896(SB), X6
	PSRAL $8, X6              // [v0, v1, 0, 0]
	// pack and interleave
	MOVO X6, X0               // temp, row0 consumed
	PACKSSLW X0, X5           // words [u0,u1,0,0, v0,v1,0,0]
	MOVO X5, X0
	PACKUSWB X0, X5           // bytes [u0,u1,0,0, v0,v1,0,0, ...]
	PSHUFB ·uvShuf(SB), X5    // [u0,v0,u1,v1]
	MOVD X5, (DI)
	ADDL $16, SI
	ADDL $16, DX
	ADDL $4, DI
	DECL CX
	JNZ uvloop
uvdone:
	RET
