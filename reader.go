package toytlv

import (
	"io"
)

// Reads TLV records from a stream.
// Note that TLVReader is buffered, i.e. it reads ahead.
// When doing Seek() on a file, recreate TLVReader, that is cheap.
type TLVReader struct {
	pre    []byte
	Reader io.Reader
}

const DefaultPreBufLength = 4096
const MinRecommendedRead = 512
const MinRecommendedWrite = 512

func ReadTLV(rdr io.Reader) *TLVReader {
	return &TLVReader{
		Reader: rdr,
		pre:    make([]byte, 0, DefaultPreBufLength),
	}
}

func (r *TLVReader) read(tolen int) error {
	l := len(r.pre)
	c := cap(r.pre)
	if c-l < MinRecommendedRead || c < tolen {
		newcap := DefaultPreBufLength
		if newcap < tolen {
			newcap = tolen
		}
		newpre := make([]byte, newcap)
		copy(newpre, r.pre)
		newpre = newpre[:l]
		r.pre = newpre
		l = len(r.pre)
		c = cap(r.pre)
	}
	for len(r.pre) < tolen {
		vac := r.pre[l:c]
		n, err := r.Reader.Read(vac)
		if err != nil {
			return err
		}
		r.pre = r.pre[0 : l+n]
	}
	return nil
}

func (r *TLVReader) ReadRecord() (lit byte, body []byte, err error) {
	var hdrlen, bodylen int
	lit, hdrlen, bodylen = ProbeHeader(r.pre)
	for lit == 0 || hdrlen+bodylen > len(r.pre) {
		tolen := len(r.pre) + 1
		if lit != 0 {
			tolen = hdrlen + bodylen
		}
		err = r.read(tolen)
		if err != nil {
			return 0, nil, err
		}
		lit, hdrlen, bodylen = ProbeHeader(r.pre)
	}
	if lit == '-' {
		return 0, nil, ErrBadRecord
	}
	body = r.pre[hdrlen : hdrlen+bodylen]
	r.pre = r.pre[hdrlen+bodylen:]
	return
}

type TLVWriter struct {
	buf    []byte
	Writer io.Writer
	Manual bool
}

func (w *TLVWriter) writeOnce() error {
	n, err := w.Writer.Write(w.buf)
	w.buf = w.buf[n:]
	return err
}

func (w *TLVWriter) WriteRecord(lit byte, body []byte) error { // todo...
	w.buf = AppendHeader(w.buf, lit, len(body))
	if len(body) >= 512 && !w.Manual { // large records: direct write
		err := w.Flush()
		if err != nil {
			return err
		}
		for len(body) > 0 {
			n, err := w.Writer.Write(body)
			if err != nil {
				return err
			}
			body = body[n:]
		}
	} else { // small records: accumulate
		w.buf = append(w.buf, body...)
		if !w.Manual && len(w.buf) >= MinRecommendedWrite {
			return w.writeOnce()
		}
	}
	return nil
}

func (w *TLVWriter) Flush() error {
	for len(w.buf) > 0 {
		err := w.writeOnce()
		if err != nil {
			return err
		}
	}
	return nil
}
