package toytlv

import (
	"sync"
	"testing"
)
import "github.com/stretchr/testify/assert"

// 1. create a server, create a client, echo
// 2. create a server, client, connect, disconn, reconnect
// 3. create a server, client, conn, stop the serv, relaunch, reconnect

type Record struct {
	lit  byte
	body []byte
	addr string
}

type TestConsumer struct {
	rcvd []Record
	mx   sync.Mutex
	co   sync.Cond
}

func (c *TestConsumer) Consume(lit byte, body []byte, address string) error {
	c.mx.Lock()
	c.rcvd = append(c.rcvd, Record{lit, body, address})
	c.co.Signal()
	c.mx.Unlock()
	return nil
}

func (c *TestConsumer) WaitDrain() (rec Record) {
	c.mx.Lock()
	if len(c.rcvd) == 0 {
		c.co.Wait()
	}
	rec = c.rcvd[0]
	c.rcvd = c.rcvd[1:]
	c.mx.Unlock()
	return rec
}

func TestTCPDepot_Connect(t *testing.T) {
	tc := TestConsumer{}
	tc.co.L = &tc.mx
	depot := TCPDepot{}
	depot.Open(&tc)

	loop := "127.0.0.1:1234"

	err := depot.Listen(loop)
	assert.Nil(t, err)

	err = depot.Connect(loop)
	assert.Nil(t, err)

	// send a record
	err = depot.Feed('M', []byte("Hi there"), loop)
	rec := tc.WaitDrain()
	assert.Equal(t, uint8('M'), rec.lit)
	assert.Equal(t, "Hi there", string(rec.body))

	// respond to that
	err = depot.Feed('M', []byte("Re: Hi there"), rec.addr)
	rerec := tc.WaitDrain()
	assert.Equal(t, uint8('M'), rerec.lit)
	assert.Equal(t, "Re: Hi there", string(rerec.body))

}
