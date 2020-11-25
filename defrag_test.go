package defrag

import (
	"errors"
	"net"
	"testing"
)

type PacketConnMock struct {
	frame  int
	packet int
	packets int
}

func (c *PacketConnMock) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if c.packet > c.packets {
		return 0, net.Addr{}, errors.New("EOF")
	}

	packetHeader := &PacketHeader {
		frameID: c.frame,
		count:   c.packet,
		number:  3,
		length:  1,
	}
	_ := binary.Write(p, binary.LittleEndian, header)
	 p[PacketHeaderSize:] = c.packet
	c.packet += 1
	return (PacketHeaderSize+1), net.Addr{}, nil

}

func Test_Read(t *testing.T) {
}
