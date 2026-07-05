package weights

import "context"

type Kind int

const (
	KindRaw Kind = iota
	KindWide
)

func (k Kind) String() string {
	switch k {
	case KindRaw:
		return "raw"
	case KindWide:
		return "wide"
	default:
		return "unknown"
	}
}

type Format int

const (
	FormatGGUF Format = iota
	FormatSafeTensors
)

func (f Format) String() string {
	switch f {
	case FormatGGUF:
		return "gguf"
	case FormatSafeTensors:
		return "safetensors"
	default:
		return "unknown"
	}
}

type Candidate struct {
	ModelID     string
	Filename    string
	DownloadURL string
	SHA256      string
	SizeBytes   int64
}

type Provider interface {
	Name() string
	Search(ctx context.Context, query string, limit int) ([]Candidate, error)
}
