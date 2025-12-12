package constants

const (
    BytesPerPixel = 4
    FormatBGRA    = "BGRA"
    FormatRGBA    = "RGBA"
)

type ResolutionPreset string

const (
    Res320p  ResolutionPreset = "320p"
    Res420p  ResolutionPreset = "420p"
    Res720p  ResolutionPreset = "720p"
    Res1080p ResolutionPreset = "1080p"
    ResOriginal ResolutionPreset = "original"
)