package toytlv

import (
	"github.com/learn-decentralized-systems/toyqueue"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestFile(t *testing.T) {
	_ = os.Remove("tmp")
	file, err := CreateFile("tmp", 1024)
	assert.Nil(t, err)
	drain, err := file.Drainer()
	assert.Nil(t, err)
	test := []byte("test")

	err = drain.Drain(toyqueue.Records{Record('T', test)})
	assert.Nil(t, err)
	err = drain.Drain(toyqueue.Records{Record('T', test, test)})
	assert.Nil(t, err)
	err = drain.Drain(toyqueue.Records{Record('T', test, test, test)})
	assert.Nil(t, err)

	feed, err := file.Feeder()
	assert.Nil(t, err)
	recs, err := feed.Feed()
	assert.Nil(t, err)
	assert.Equal(t, 3, len(recs))
	assert.Equal(t, 6, len(recs[0]))
	assert.Equal(t, 10, len(recs[1]))
	assert.Equal(t, 14, len(recs[2]))

	recs, err = feed.Feed()
	assert.Equal(t, 0, len(recs))
	assert.Equal(t, ErrBadRecord, err)

	_ = os.Remove("tmp")
}
