package toytlv

import (
	"encoding/binary"
	"io"
)

// Reads TLV records from a stream; Peek() provides the record type [A-Z]
// and the (remaining) record length. Read() reads no more than the
// current record (and no less, if the space if provided).
//
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
	// while <5 <2 bytes: read full buf if avail
	l := len(r.pre)
	for l == 0 || (TLVLongLit(r.pre[0]) && l < 5) || (TLVShortLit(r.pre[0]) && l < 2) {
		err = r.read(2)
		if err != nil {
			return 0, nil, err
		}
		l = len(r.pre)
	}
	var lenlen int
	var bodylen int
	if TLVLongLit(r.pre[0]) {
		lenlen = 4
		readlen := binary.LittleEndian.Uint32(r.pre[1:5])
		if readlen > 0x7fffffff {
			return 0, nil, ErrBadRecord
		}
		bodylen = int(readlen)
		lit = r.pre[0]
	} else if TLVShortLit(r.pre[0]) {
		lenlen = 1
		bodylen = int(r.pre[1])
		lit = r.pre[0] - ('a' - 'A')
	} else {
		return 0, nil, ErrBadRecord
	}
	fullen := 1 + lenlen + bodylen
	for len(r.pre) < fullen {
		err = r.read(fullen)
		if err != nil {
			return
		}
	}
	body = r.pre[1+lenlen : 1+lenlen+bodylen]
	r.pre = r.pre[1+lenlen+bodylen:]
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

func (w *TLVWriter) WriteRecord(lit byte, body []byte) error {
	if !TLVLongLit(lit) {
		return ErrBadRecord
	}
	if len(body) <= 0xff {
		w.buf = append(w.buf, lit+('a'-'A'))
		w.buf = append(w.buf, byte(len(body)))
	} else {
		var lenbuf = []byte{0, 0, 0, 0}
		binary.LittleEndian.PutUint32(lenbuf, uint32(len(body)))
		w.buf = append(w.buf, lit)
		w.buf = append(w.buf, lenbuf...)
	}
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
