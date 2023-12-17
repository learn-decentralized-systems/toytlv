package toytlv

import (
	"encoding/binary"
	"errors"
)

func TLVShortLit(lit byte) bool {
	return lit >= 'a' && lit <= 'z'
}

func TLVLongLit(lit byte) bool {
	return lit >= 'A' && lit <= 'Z'
}

func TLVlit(lit byte) bool {
	return TLVShortLit(lit) || TLVLongLit(lit)
}

var ErrIncomplete = errors.New("incomplete data")
var ErrBadRecord = errors.New("bad TLV record format")

// Takes a TLV record from the slice
func Drain(data *[]byte) (lit byte, body []byte, err error) {
	if len(*data) < 2 {
		err = ErrIncomplete
		return
	}
	lit = (*data)[0]
	if TLVShortLit(lit) {
		lit = lit - ('a' - 'A')
		reclen := int((*data)[1])
		if len(*data) < 2+reclen {
			err = ErrIncomplete
		} else {
			body = (*data)[2 : 2+reclen]
			*data = (*data)[2+reclen:]
		}
	} else if TLVLongLit(lit) {
		if len(*data) < 5 {
			err = ErrIncomplete
			return
		}
		reclen := binary.LittleEndian.Uint32((*data)[1:5])
		if reclen > 1<<30 {
			err = ErrBadRecord
		} else if len(*data) < 5+int(reclen) {
			err = ErrIncomplete
		} else {
			body = (*data)[5 : 5+reclen]
			*data = (*data)[5+reclen:]
		}
	} else {
		err = ErrBadRecord
	}
	return
}

func Feed(data *[]byte, lit byte, body []byte) {
	if !TLVLongLit(lit) {
		panic("TLV liters are uppercase A-Z")
	}
	blen := len(body)
	if blen < 0x100 {
		*data = append(*data, lit+('a'-'A'), uint8(blen))
	} else {
		i := [4]byte{}
		binary.LittleEndian.PutUint32(i[:], uint32(blen))
		*data = append(*data, lit)
		*data = append(*data, i[:]...)
	}
	*data = append(*data, body...)
	return
}

func TLVAppend2(data []byte, lit byte, body1, body2 []byte) (newdata []byte, err error) {
	if !TLVLongLit(lit) {
		return nil, ErrBadRecord
	}
	blen := len(body1) + len(body2)
	i := [4]byte{}
	binary.LittleEndian.PutUint32(i[:], uint32(blen))
	if blen < 255 {
		newdata = append(data, lit+('a'-'A'), i[0])
		newdata = append(newdata, body1...)
		newdata = append(newdata, body2...)
	} else {
		newdata = append(data, lit)
		newdata = append(newdata, i[:]...)
		newdata = append(newdata, body1...)
		newdata = append(newdata, body2...)
	}
	return
}

// FeedHeader opens a streamed TLV record; use append() to create the
// record body, then call CloseHeader(&buf, bookmark)
func FeedHeader(buf *[]byte, lit byte) (bookmark int) {
	if !TLVLongLit(lit) {
		panic("TLV liters are uppercase A-Z")
	}
	*buf = append(*buf, lit)
	blanclen := []byte{0, 0, 0, 0}
	*buf = append(*buf, blanclen...)
	return len(*buf)
}

// CloseHeader closes a streamed TLV record
func CloseHeader(buf *[]byte, bookmark int) {
	if bookmark < 5 || len(*buf) < bookmark {
		panic("check the API docs")
	}
	binary.LittleEndian.PutUint32((*buf)[bookmark-4:bookmark], uint32(len(*buf)-bookmark))
}
