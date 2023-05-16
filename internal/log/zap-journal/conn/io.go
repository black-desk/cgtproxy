package conn

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"go.uber.org/zap"
)

var _ zap.Sink = &JournalConn{}

func (c *JournalConn) Write(p []byte) (n int, err error) {
	if c.limit != -1 && len(p) >= int(c.limit) {
		return c.writeWithMemFD(p)
	}
	n, _, err = c.unixConn.WriteMsgUnix(p, nil, c.unixAddr)
	return
}

func (c *JournalConn) writeWithMemFD(p []byte) (n int, err error) {
	var file *os.File
	file, err = ioutil.TempFile("/dev/shm/", "journal.XXXXX")
	defer file.Close()
	if err != nil {
		return
	}

	err = syscall.Unlink(file.Name())
	if err != nil {
		return
	}

	_, err = io.Copy(file, bytes.NewBuffer(p))
	if err != nil {
		return
	}

	rights := syscall.UnixRights(int(file.Fd()))
	_, _, err = c.unixConn.WriteMsgUnix([]byte{}, rights, c.unixAddr)
	if err != nil {
		return
	}

	return
}

func (c *JournalConn) Sync() error {
	return nil
}

func (c *JournalConn) Close() error {
	return c.unixConn.Close()
}
