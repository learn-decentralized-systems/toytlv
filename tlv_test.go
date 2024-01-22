package toytlv

import (
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestTLVAppend(t *testing.T) {
	buf := []byte{}
	buf = Append(buf, 'A', []byte{'A'})
	buf = Append(buf, 'b', []byte{'B', 'B'})
	correct2 := []byte{'a', 1, 'A', '2', 'B', 'B'}
	assert.Equal(t, correct2, buf, "basic TLV fail")

	var c256 [256]byte
	for n, _ := range c256 {
		c256[n] = 'c'
	}
	buf = Append(buf, 'C', c256[:])
	assert.Equal(t, len(correct2)+1+4+len(c256), len(buf))
	assert.Equal(t, uint8(67), buf[len(correct2)])
	assert.Equal(t, uint8(1), buf[len(correct2)+2])

	lit, body, buf, err := TakeAnyWary(buf)
	assert.Nil(t, err)
	assert.Equal(t, uint8('A'), lit)
	assert.Equal(t, []byte{'A'}, body)

	body2, buf, err2 := TakeWary('B', buf)
	assert.Nil(t, err2)
	assert.Equal(t, []byte{'B', 'B'}, body2)
}

func TestFeedHeader(t *testing.T) {
	buf := []byte{}
	l, buf := OpenHeader(buf, 'A')
	text := "some text"
	buf = append(buf, text...)
	CloseHeader(buf, l)
	lit, body, rest, err := TakeAnyWary(buf)
	assert.Nil(t, err)
	assert.Equal(t, uint8('A'), lit)
	assert.Equal(t, text, string(body))
	assert.Equal(t, 0, len(rest))
}

func TestTLVReader_ReadRecord(t *testing.T) {
	const K = 1000
	const L = 512
	_ = os.Remove("tlv")
	file, err := os.OpenFile("tlv", os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	assert.Nil(t, err)
	writer := TLVWriter{
		Writer: file,
	}
	var lo [L]byte
	for i := 0; i < L; i++ {
		lo[i] = byte(i)
	}
	var sho = [1]byte{'A'}
	for i := 0; i < K; i++ {
		err = writer.WriteRecord('L', lo[:])
		assert.Nil(t, err)
		err = writer.WriteRecord('S', sho[:])
		assert.Nil(t, err)
	}
	err = writer.Flush()
	assert.Nil(t, err)
	info, err := file.Stat()
	assert.Nil(t, err)
	assert.Equal(t, int64((2+1)*K+(5+len(lo))*K), info.Size())
	_ = file.Close()

	file2, err := os.Open("tlv")
	assert.Nil(t, err)
	reader := TLVReader{
		Reader: file2,
	}
	for i := 0; i < K; i++ {

		lit, body, err := reader.ReadRecord()
		assert.Nil(t, err)
		assert.Equal(t, byte('L'), lit)
		assert.Equal(t, lo[:], body)

		lit, body, err = reader.ReadRecord()
		assert.Nil(t, err)
		assert.Equal(t, byte('S'), lit)
		assert.Equal(t, sho[:], body)

	}

	lit, body, err := reader.ReadRecord()
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, byte(0), lit)
	assert.Equal(t, 0, len(body))

	_ = os.Remove("tlv")
}
