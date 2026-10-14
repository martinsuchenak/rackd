package model

const (
	DefaultPageSize = 100
	MaxPageSize     = 1000
)

// Pagination provides limit/offset for list queries.
// Embed in filter structs to add pagination support.
type Pagination struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// Clamp applies defaults and caps to pagination values.
func (p *Pagination) Clamp() {
	if p.Limit <= 0 {
		p.Limit = DefaultPageSize
	}
	if p.Limit > MaxPageSize {
		p.Limit = MaxPageSize
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}
