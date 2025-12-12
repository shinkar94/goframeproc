package resize

// ResizeParams contains parameters for the resize operation
type ResizeParams struct {
    Src         []byte // Source image buffer
    SrcW        int    // Width of the source image
    SrcH        int    // Height of the source image
	Stride      int    // Row stride in bytes (may differ from SrcW*4 with padding)
    TargetW     int    // Target width
    TargetH     int    // Target height
    Mode        Mode   // Interpolation mode (Nearest/Bilinear)
	Format      string // Output format (RGBA/BGRA)
    InputFormat string // Input format (RGBA/BGRA)
    Dst         []byte // Target buffer (must be pre-allocated)
}

// determines the resize execution strategy
type ResizeStrategy interface {
	DoResize(params ResizeParams, tables *LookupTables)
}

// single-threaded strategy, saves CPU
type SequentialResizeStrategy struct{}

// multi-threaded strategy, maximum speed
type ParallelResizeStrategy struct{}
