package toytlv

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

type Consumer interface {
	Consume(lit byte, body []byte, address string) error
}

type TCPConn struct {
	depot *TCPDepot
	addr  string
	conn  net.Conn
	out   []byte
	outmx sync.Mutex
}

type TCPDepot struct {
	conns   map[string]*TCPConn
	listens map[string]net.Listener
	conmx   sync.Mutex
	in      Consumer
}

func (de *TCPDepot) Open(in Consumer) {
	de.conmx.Lock()
	de.conns = make(map[string]*TCPConn)
	de.listens = make(map[string]net.Listener)
	de.conmx.Unlock()
	de.in = in
}

func (de *TCPDepot) Close() {
	for _, lstn := range de.listens {
		_ = lstn.Close()
	}
	de.listens = nil
	for _, con := range de.conns {
		con.Close()
	}
	de.conmx.Lock()
	de.conns = make(map[string]*TCPConn)
	de.listens = make(map[string]net.Listener)
	de.conmx.Unlock()
}

func (tcp *TCPConn) Close() {
	// TODO writer closes on complete | 1 sec expired
	c := tcp.conn
	_ = c.Close()
}

var ErrAddressUnknown = errors.New("address unknown")

func (tcp *TCPDepot) Consume(lit byte, body []byte, address string) error {
	conn, ok := tcp.conns[address]
	if !ok {
		return ErrAddressUnknown
	}
	conn.outmx.Lock()
	var len [4]byte // FIXME ToyTLV
	conn.out = append(conn.out, lit)
	conn.out = append(conn.out, len[0:4]...)
	conn.out = append(conn.out, body...)
	conn.outmx.Unlock()
	return nil
}

const MAX_RETRY_PERIOD = time.Minute
const MIN_RETRY_PERIOD = time.Second / 2

// attrib?!
func (de *TCPDepot) Connect(addr string) (err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	peer := TCPConn{
		depot: de,
		conn:  conn,
		addr:  addr,
	}
	de.conmx.Lock()
	de.conns[addr] = &peer
	de.conmx.Unlock()
	go peer.KeepTalking()
	return nil
}

var ErrDisconnected = errors.New("disconnected by user")

func (tcp *TCPConn) KeepTalking() {
	talk_backoff := MIN_RETRY_PERIOD
	conn_backoff := MIN_RETRY_PERIOD
	for {

		conntime := time.Now()
		go tcp.doWrite()
		err := tcp.Read()
		tcp.conn = nil

		if err == ErrDisconnected {
			return
		}

		atLeast5min := conntime.Add(time.Minute * 5)
		if atLeast5min.After(time.Now()) {
			talk_backoff *= 2 // connected, tried to talk, failed => wait more
			if talk_backoff > MAX_RETRY_PERIOD {
				talk_backoff = MAX_RETRY_PERIOD
			}
		}

		for tcp.conn == nil {
			time.Sleep(conn_backoff + talk_backoff)
			tcp.conn, err = net.Dial("tcp", tcp.addr)
			if err != nil {
				conn_backoff = conn_backoff * 2
				if conn_backoff > MAX_RETRY_PERIOD/2 {
					conn_backoff = MAX_RETRY_PERIOD
				}
			} else {
				conn_backoff = MIN_RETRY_PERIOD
			}
		}

	}
}

func (de *TCPDepot) Feed(lit byte, body []byte, addr string) error {
	if !TLVLongLit(lit) {
		panic("TLV litera must be A-Z")
	}
	de.conmx.Lock()
	conn, ok := de.conns[addr]
	de.conmx.Unlock()
	if !ok {
		return ErrAddressUnknown
	}
	conn.outmx.Lock()
	Feed(&conn.out, lit, body)
	conn.outmx.Unlock()
	return nil
}

func (de *TCPDepot) Disconnect(addr string) (err error) {
	de.conmx.Lock()
	tcp, ok := de.conns[addr]
	de.conmx.Unlock()
	if !ok {
		return ErrAddressUnknown
	}
	tcp.conn = nil
	de.conmx.Lock()
	delete(de.conns, addr)
	de.conmx.Unlock()
	return nil
}

func (de *TCPDepot) Listen(addr string) (err error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen() fails: %s\r\n", err.Error())
		return
	}
	de.conmx.Lock()
	pre, ok := de.listens[addr]
	if ok {
		_ = pre.Close()
	}
	de.listens[addr] = listener
	de.conmx.Unlock()
	go de.KeepListening(addr)
	return
}

func (de *TCPDepot) StopListening(addr string) error {
	de.conmx.Lock()
	listener, ok := de.listens[addr]
	delete(de.listens, addr)
	de.conmx.Unlock()
	if !ok {
		return ErrAddressUnknown
	}
	return listener.Close()
}

func (de *TCPDepot) KeepListening(addr string) {
	for {
		de.conmx.Lock()
		listener, ok := de.listens[addr]
		de.conmx.Unlock()
		if !ok {
			break
		}
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err.Error())
			break
		}
		addr := conn.RemoteAddr().String()
		fmt.Fprintf(os.Stderr, "%s connected\r\n", addr)
		peer := TCPConn{
			depot: de,
			conn:  conn,
			addr:  addr,
		}
		de.conmx.Lock()
		de.conns[addr] = &peer
		de.conmx.Unlock()

		go peer.doWrite()
		go peer.doRead()

	}
}

func (tcp *TCPConn) doRead() {
	err := tcp.Read()
	if err != nil && err != ErrDisconnected {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
	}
}

func (tcp *TCPConn) doWrite() {
	conn := tcp.conn
	for conn != nil {
		tcp.outmx.Lock()
		out := tcp.out
		tcp.outmx.Unlock()
		if len(out) == 0 {
			time.Sleep(time.Millisecond) // FIXME condition!!!
			continue
		}
		n, err := conn.Write(out)
		//fmt.Fprintf(os.Stderr, "sent %d bytes\n", n)
		if err != nil {
			tcp.conn = nil
			_, _ = fmt.Fprint(os.Stderr, err.Error())
			break
		}
		tcp.outmx.Lock()
		tcp.out = tcp.out[n:]
		tcp.outmx.Unlock()
		conn = tcp.conn
	}
}

func (tcp *TCPConn) Read() (err error) {
	var buf []byte
	var lit byte
	var body []byte
	conn := tcp.conn
	for conn != nil {
		buf, err = ReadBuf(buf, conn)
		//fmt.Fprintf(os.Stderr, "bytes pending %d\n", len(buf))
		if err != nil {
			break
		}
		lit, body, err = Drain(&buf)
		if err == ErrIncomplete {
			time.Sleep(time.Millisecond)
			continue
		} else if err != nil {
			break
		}

		err = tcp.depot.in.Consume(lit, body, tcp.addr)

		if err != nil {
			break
		}
		conn = tcp.conn
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n\r", err.Error())
	}

	tcp.conn = nil
	return
}

func ReadBuf(buf []byte, rdr io.Reader) ([]byte, error) {
	avail := cap(buf) - len(buf)
	if avail < 512 {
		l := 4096
		if len(buf) > 2048 {
			l = len(buf) * 2
		}
		newbuf := make([]byte, l)
		copy(newbuf[:], buf)
		buf = newbuf[:len(buf)]
	}
	idle := buf[len(buf):cap(buf)]
	n, err := rdr.Read(idle)
	if err != nil {
		return buf, nil
	}
	buf = buf[:len(buf)+n]
	return buf, nil
}
