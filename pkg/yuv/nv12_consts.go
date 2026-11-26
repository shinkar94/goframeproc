package yuv

// SSSE3/SSE4.1 kernel constants shared by the amd64 and 386 assemblies.
// The kernels reference these as memory operands — the 386 twin has only
// X0-X7, so constants never occupy registers.
//
// Y path (per 4 px): pshufb gathers (r,g) and (b,0) byte pairs, PMOVZXBW
// widens (src != dst: same-register widen is unreliable), PMADDWD (s16*s16
// -> s32) applies BT.601 weights exactly: 66*r + 129*g and 25*b. The
// rounding folds into one add: y = (sum + 128 + 16<<8) >> 8 = (sum+dY4224)>>8.
var (
	yMaskRG = [16]byte{2, 1, 6, 5, 10, 9, 14, 13, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	yMaskB0 = [16]byte{0, 0x80, 4, 0x80, 8, 0x80, 12, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	yMadRG  = [8]int16{66, 129, 66, 129, 66, 129, 66, 129}
	yMadB   = [8]int16{25, 0, 25, 0, 25, 0, 25, 0}
	dY4224  = [4]uint32{4224, 4224, 4224, 4224}
)

// UV path (per 16-byte lane = 4 px per row = 2 chroma blocks): uvPair
// gathers the channel pairs of both rows, PMADDUBSW with uvOnes sums each
// channel over its 2x2 block, uvTwo+PSRLW 2 finishes the rounded box mean,
// uvRG/uvMB reorder means to (mr,mg)/(mb,0) pairs for PMADDWD, uvShuf
// interleaves [u0,v0,u1,v1] after the packs. u/v fold their +128<<1
// rounding the same way: (part + dUV32896) >> 8.
var (
	uvPair = [16]byte{0, 4, 1, 5, 2, 6, 8, 12, 9, 13, 10, 14, 0x80, 0x80, 0x80, 0x80}
	uvOnes = [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	uvTwo  = [8]int16{2, 2, 2, 2, 2, 2, 2, 2}
	// pmaddubsw word layout: [Σb0,Σg0,Σr0, Σb1,Σg1,Σr1, 0,0] — block 1
	// occupies words 3-5, hence the byte offsets below.
	uvRG     = [16]byte{4, 5, 2, 3, 10, 11, 8, 9, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	uvMB     = [16]byte{0, 1, 0x80, 0x80, 6, 7, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	mU1      = [8]int16{-38, -74, -38, -74, -38, -74, -38, -74}
	mV1      = [8]int16{112, -94, 112, -94, 112, -94, 112, -94}
	mU2      = [8]int16{112, 0, 112, 0, 112, 0, 112, 0}
	mV2      = [8]int16{-18, 0, -18, 0, -18, 0, -18, 0}
	dUV32896 = [4]uint32{32896, 32896, 32896, 32896}
	// after PACKSSLW+PACKUSWB the bytes are [u0,u1,0,0, v0,v1,0,0]:
	// interleave to [u0,v0,u1,v1].
	uvShuf = [16]byte{0, 4, 1, 5, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
)
