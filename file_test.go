package toytlv

import (
	"github.com/learn-decentralized-systems/toyqueue"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestFile(t *testing.T) {
	_ = os.Remove("tmp")
	file, err := CreateFile("tmp", 10000)
	assert.Nil(t, err)
	drain, err := file.Drainer()
	assert.Nil(t, err)
	test := []byte("test")
	bigbuf := make([]byte, 0, 8192)
	for cap(bigbuf) > len(bigbuf) {
		bigbuf = append(bigbuf, test...)
	}

	err = drain.Drain(toyqueue.Records{Record('T', test)})
	assert.Nil(t, err)
	err = drain.Drain(toyqueue.Records{Record('T', test, test)})
	assert.Nil(t, err)
	err = drain.Drain(toyqueue.Records{Record('T', test, test, test)})
	assert.Nil(t, err)
	err = drain.Drain(toyqueue.Records{Record('T', bigbuf)})
	assert.Nil(t, err)

	feed, err := file.Feeder()
	assert.Nil(t, err)
	recs, err := feed.Feed()
	assert.Nil(t, err)
	assert.Equal(t, 3, len(recs)) // 4K default page
	assert.Equal(t, 6, len(recs[0]))
	assert.Equal(t, 10, len(recs[1]))
	assert.Equal(t, 14, len(recs[2]))
	recs, err = feed.Feed()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(recs)) // must complete the record

	recs, err = feed.Feed()
	assert.Equal(t, 0, len(recs))
	assert.Equal(t, ErrBadRecord, err)

	pos, err := feed.Seek(6, io.SeekStart)
	assert.Nil(t, err)
	assert.Equal(t, int64(6), pos)
	recs, err = feed.Feed()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(recs))
	assert.Equal(t, 10, len(recs[0]))
	assert.Equal(t, 14, len(recs[1]))

	_ = os.Remove("tmp")
}
