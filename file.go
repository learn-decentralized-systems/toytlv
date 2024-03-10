package toytlv

import (
	"github.com/learn-decentralized-systems/toyqueue"
	"golang.org/x/sys/unix"
	"io"
	"os"
)

// File is (you guess it) a Unix file. To allow for parallel
// reading/writing from/to many independent positions, all
// reading/writing goes through separate Feeders/Drainers.
type File struct {
	fd int
}

type fileFeeder struct {
	file *File
	pos  int64
	rest []byte
}

type fileDrainer struct {
	file *File
	pos  int64
}

// Feeder, thread-unsafe, for this file. Very cheap.
func (f *File) Feeder() toyqueue.FeedSeekCloser {
	return &fileFeeder{file: f}
}

func (f *File) Drainer() toyqueue.DrainSeekCloser {
	return &fileDrainer{file: f}
}

func (file *File) Open(path string, mode int, perm uint32) (err error) {
	file.fd, err = unix.Open(path, mode, perm)
	return
}

// create an empty TLV file
func CreateFile(path string, size int64) (file *File, err error) {
	file = &File{}
	err = file.Open(path, unix.O_CREAT|unix.O_RDWR, 0660)
	if err == nil && size > 0 {
		err = unix.Ftruncate(file.fd, size)
	}
	return
}

// open a TLV file
func OpenFile(path string) (file *File, err error) {
	file = &File{}
	err = file.Open(path, unix.O_RDWR, 0)
	return
}

func OpenFileReadOnly(path string) (file *File, err error) {
	file = &File{}
	err = file.Open(path, unix.O_RDONLY, 0)
	return
}

func (f *File) Size() int64 {
	var stat unix.Stat_t
	_ = unix.Fstat(f.fd, &stat)
	return stat.Size
}

func (f *File) fdesc() int {
	if f == nil {
		return -1
	}
	return f.fd
}

func (f *File) Sync() (err error) {
	if f.fd == -1 {
		return os.ErrClosed
	}
	err = unix.Fsync(f.fd)
	return
}

func (f *File) Close() (err error) {
	if f.fd == -1 {
		return os.ErrClosed
	}
	err = unix.Close(f.fd)
	if err == nil {
		f.fd = -1
	}
	return
}

func (f *fileDrainer) Drain(recs toyqueue.Records) (err error) {
	fd := f.file.fdesc()
	if fd == -1 {
		return os.ErrClosed
	}
	n := 0
	for len(recs) > 0 && err == nil {
		n, err = unix.Writev(fd, recs)
		recs = recs.ExactSuffix(int64(n))
	}
	return
}

// If the position is out of range, returns no error (Drain will return)
func (ff *fileDrainer) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart: // TODO checks
		ff.pos = offset
	case io.SeekCurrent:
		ff.pos += offset
	case io.SeekEnd:
		sz := ff.file.Size()
		ff.pos = sz - offset
	default:
		return -1, os.ErrInvalid
	}
	return ff.pos, nil
}

func (ff *fileDrainer) Close() error {
	if ff.file == nil {
		return os.ErrClosed
	}
	ff.file = nil
	ff.pos = 0
	return nil
}

// can return empty recs if e.g. there is a laaarge incomplete incoming record
// On EoF returns err==io.EOF
func (ff *fileFeeder) Feed() (recs toyqueue.Records, err error) {
	fdr := fdReader{fd: ff.file.fd, pos: ff.pos}
	more := 512 // min disk sector
	if len(ff.rest) > 0 {
		// here we trust the file that it will not DDoS us with 2GB headers
		// can not do the same with the network
		inc := Incomplete(ff.rest)
		if inc > more {
			more = inc
		}
	}
	ff.rest, err = AppendRead(ff.rest, &fdr, more)
	if err == nil {
		ff.pos = fdr.pos
		recs, ff.rest, err = Split(ff.rest)
	} else if err == io.EOF && len(ff.rest) > 0 {
		recs, ff.rest, err = Split(ff.rest)
	}
	return
}

func (ff *fileFeeder) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart: // TODO checks
		ff.pos = offset
	case io.SeekCurrent:
		ff.pos += offset
	case io.SeekEnd:
		sz := ff.file.Size()
		ff.pos = sz - offset
	default:
		return -1, os.ErrInvalid
	}
	ff.rest = nil
	return ff.pos, nil
}

func (ff *fileFeeder) Close() error {
	if ff.file == nil {
		return os.ErrClosed
	}
	ff.file = nil
	ff.rest = nil
	ff.pos = 0
	return nil
}

type fdReader struct {
	fd  int
	pos int64
}

func (fd *fdReader) Read(into []byte) (n int, err error) {
	n, err = unix.Pread(fd.fd, into, fd.pos)
	if n > 0 {
		fd.pos += int64(n)
	}
	return
}
