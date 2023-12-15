package toytlv

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTLVAppend(t *testing.T) {
	buf := []byte{}
	Feed(&buf, 'A', []byte{'A'})
	Feed(&buf, 'B', []byte{'B', 'B'})
	correct2 := []byte{'a', 1, 'A', 'b', 2, 'B', 'B'}
	assert.Equal(t, correct2, buf, "basic TLV fail")

	var c256 [256]byte
	for n, _ := range c256 {
		c256[n] = 'c'
	}
	Feed(&buf, 'C', c256[:])
	assert.Equal(t, len(correct2)+1+4+len(c256), len(buf))
	assert.Equal(t, uint8(67), buf[len(correct2)])
	assert.Equal(t, uint8(1), buf[len(correct2)+2])

	lit, body, err := Drain(&buf)
	assert.Nil(t, err)
	assert.Equal(t, uint8('A'), lit)
	assert.Equal(t, []byte{'A'}, body)

	lit2, body2, err2 := Drain(&buf)
	assert.Nil(t, err2)
	assert.Equal(t, uint8('B'), lit2)
	assert.Equal(t, []byte{'B', 'B'}, body2)
}

func TestFeedHeader(t *testing.T) {
	buf := []byte{}
	l := FeedHeader(&buf, 'A')
	text := "some text"
	buf = append(buf, text...)
	CloseHeader(&buf, l)
	lit, body, err := Drain(&buf)
	assert.Nil(t, err)
	assert.Equal(t, uint8('A'), lit)
	assert.Equal(t, text, string(body))
}
