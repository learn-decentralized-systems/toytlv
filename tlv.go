package toytlv

import (
	"encoding/binary"
	"errors"
	"github.com/learn-decentralized-systems/toylog/toyqueue"
)

const CaseBit uint8 = 'a' - 'A'

var ErrIncomplete = errors.New("incomplete data")
var ErrBadRecord = errors.New("bad TLV record format")

// ProbeHeader probes a TLV record header. Return values:
//   - 0  0 0 	incomplete header
//   - '-' 0 0 	bad format
//   - 'A' 2 123 success
func ProbeHeader(data []byte) (lit byte, hdrlen, bodylen int) {
	if len(data) == 0 {
		return 0, 0, 0
	}
	dlit := data[0]
	if dlit >= '0' && dlit <= '9' { // tiny
		lit = '0'
		bodylen = int(dlit - '0')
		hdrlen = 1
	} else if dlit >= 'a' && dlit <= 'z' { // short
		if len(data) < 2 {
			return
		}
		lit = dlit - CaseBit
		hdrlen = 2
		bodylen = int(data[1])
	} else if dlit >= 'A' && dlit <= 'Z' { // long
		if len(data) < 5 {
			return
		}
		bl := binary.LittleEndian.Uint32(data[1:5])
		if bl > 0x7fffffff {
			lit = '-'
			return
		}
		lit = dlit
		bodylen = int(bl)
		hdrlen = 5
	} else {
		lit = '-'
	}
	return
}

func Split(data []byte) (recs toyqueue.Records, rest []byte, err error) {
	rest = data
	for len(rest) > 0 {
		lit, hlen, blen := ProbeHeader(rest)
		if lit == '-' {
			err = ErrBadRecord
			return
		}
		if lit == 0 {
			return
		}
		recs = append(recs, rest[:hlen+blen])
		rest = rest[hlen+blen:]
	}
	return
}

func ProbeHeaders(lits string, data []byte) int {
	rest := data
	for i := 0; i < len(lits); i++ {
		l, hl, bl := ProbeHeader(rest)
		if l != lits[i] {
			return -1
		}
		rest = rest[hl+bl:]
	}
	return len(data) - len(rest)
}

// Feeds the header into the buffer.
// Subtle: lower-case lit allows for defaulting, uppercase must be explicit.
func AppendHeader(into []byte, lit byte, bodylen int) (ret []byte) {
	biglit := lit &^ CaseBit
	if biglit < 'A' || biglit > 'Z' {
		panic("ToyTLV record type is A..Z")
	}
	if bodylen < 10 && (lit&CaseBit) != 0 {
		ret = append(into, byte('0'+bodylen))
	} else if bodylen > 0xff {
		if bodylen > 0x7fffffff {
			panic("oversized TLV record")
		}
		ret = append(into, biglit)
		ret = binary.LittleEndian.AppendUint32(ret, uint32(bodylen))
	} else {
		ret = append(into, lit|CaseBit, byte(bodylen))
	}
	return ret
}

// Take is used to read safe TLV inputs (e.g. from own storage) with
// record types known in advance.
func Take(lit byte, data []byte) (body, rest []byte) {
	flit, hdrlen, bodylen := ProbeHeader(data)
	if flit == 0 || hdrlen+bodylen > len(data) {
		return nil, data // Incomplete
	}
	if flit != lit && flit != '0' {
		return nil, nil // BadRecord
	}
	body = data[hdrlen : hdrlen+bodylen]
	rest = data[hdrlen+bodylen:]
	return
}

// TakeAny is used for safe TLV inputs when record types can vary.
func TakeAny(data []byte) (lit byte, body, rest []byte) {
	if len(data) == 0 {
		return 0, nil, nil
	}
	lit = data[0] & ^CaseBit
	body, rest = Take(lit, data)
	return
}

// TakeWary reads TLV records of known type from unsafe input.
func TakeWary(lit byte, data []byte) (body, rest []byte, err error) {
	flit, hdrlen, bodylen := ProbeHeader(data)
	if flit == 0 || hdrlen+bodylen > len(data) {
		return nil, data, ErrIncomplete
	}
	if flit != lit && flit != '0' {
		return nil, nil, ErrBadRecord
	}
	body = data[hdrlen : hdrlen+bodylen]
	rest = data[hdrlen+bodylen:]
	return
}

// TakeWary reads TLV records of arbitrary type from unsafe input.
func TakeAnyWary(data []byte) (lit byte, body, rest []byte, err error) {
	if len(data) == 0 {
		return 0, nil, nil, ErrIncomplete
	}
	lit = data[0] & ^CaseBit
	body, rest = Take(lit, data)
	return
}

func TakeRecord(lit byte, data []byte) (rec, rest []byte) {
	flit, hdrlen, bodylen := ProbeHeader(data)
	if flit == 0 || hdrlen+bodylen > len(data) {
		return nil, data // Incomplete
	}
	if flit != lit && flit != '0' {
		return nil, nil // BadRecord
	}
	rec = data[0 : hdrlen+bodylen]
	rest = data[hdrlen+bodylen:]
	return
}

func TakeAnyRecord(data []byte) (lit byte, rec, rest []byte) {
	lit, hdrlen, bodylen := ProbeHeader(data)
	if lit == 0 || hdrlen+bodylen > len(data) {
		return 0, nil, data // Incomplete
	}
	if lit == '-' {
		return '-', nil, nil // BadRecord
	}
	rec = data[0 : hdrlen+bodylen]
	rest = data[hdrlen+bodylen:]
	return
}

func TotalLen(inputs [][]byte) (sum int) {
	for _, input := range inputs {
		sum += len(input)
	}
	return
}

func Lit(rec []byte) byte {
	b := rec[0]
	if b >= 'a' && b <= 'z' {
		return b - CaseBit
	} else if b >= 'A' && b <= 'Z' {
		return b
	} else if b >= '0' && b <= '9' {
		return '0'
	} else {
		return '-'
	}
}

// Append appends a record to the buffer; note that uppercase type
// is always explicit, lowercase can be defaulted.
func Append(into []byte, lit byte, body ...[]byte) (res []byte) {
	total := TotalLen(body)
	res = AppendHeader(into, lit, total)
	for _, b := range body {
		res = append(res, b...)
	}
	return res
}

// Record composes a record of a given type; mainly for debugging.
func Record(lit byte, body ...[]byte) []byte {
	total := TotalLen(body)
	ret := make([]byte, 0, total+5)
	ret = AppendHeader(ret, lit, total)
	for _, b := range body {
		ret = append(ret, b...)
	}
	return ret
}

func Concat(msg ...[]byte) []byte {
	total := TotalLen(msg)
	ret := make([]byte, 0, total)
	for _, b := range msg {
		ret = append(ret, b...)
	}
	return ret
}

// OpenHeader opens a streamed TLV record; use append() to create the
// record body, then call CloseHeader(&buf, bookmark)
func OpenHeader(buf []byte, lit byte) (bookmark int, res []byte) {
	lit &= ^CaseBit
	if lit < 'A' || lit > 'Z' {
		panic("TLV liters are uppercase A-Z")
	}
	res = append(buf, lit)
	blanclen := []byte{0, 0, 0, 0}
	res = append(res, blanclen...)
	return len(res), res
}

// CloseHeader closes a streamed TLV record
func CloseHeader(buf []byte, bookmark int) {
	if bookmark < 5 || len(buf) < bookmark {
		panic("check the API docs")
	}
	binary.LittleEndian.PutUint32(buf[bookmark-4:bookmark], uint32(len(buf)-bookmark))
}
