package defrag

import (
	"testing"
)

type PacketConnMock struct {
	frame  int
	packet int
}

func (c *PacketConnMock) ReadFrom(p []byte) (n int, addr Addr, err error) {
	packetHeader := PacketHeader {
		frameID: c.frame,
		count:   c.packet,
		number:  3,
		length:  1,
	}
	
}

func Test_Read(t *testing.T) {
}
