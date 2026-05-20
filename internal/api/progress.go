package api

import "io"

// progressReader wraps an io.Reader and invokes cb after each successful Read.
type progressReader struct {
	r       io.Reader
	total   int64
	written int64
	cb      func(written, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.written += int64(n)
		if p.cb != nil {
			p.cb(p.written, p.total)
		}
	}
	return n, err
}
