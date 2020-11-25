package defrag

import (
	"errors"
	"binary"
	"net"
	"testing"
	"unsafe"

	"encoding/binary"
)

type PacketConnMock struct {
	frame   uint32
	packet  uint16
	packets uint16
}

func (c *PacketConnMock) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if c.packet > c.packets {
		return 0, nil, errors.New("EOF")
	}

	packetHeader := &PacketHeader {
		frameID: c.frame,
		count:   c.packet,
		number:  3,
		length:  1,
	}
	binary.Write(p, binary.LittleEndian, header)
	 p[PacketHeaderSize:] = c.packet
	c.packet += 1
	return (PacketHeaderSize+1), net.Addr{}, nil

}

func Test_Read(t *testing.T) {
	packetConnMock := &PacketConnMock {
		packets: 3,
	}
	reader := new(packetConnMock)
}


// Check padding of the PacketHeader
func Test_PacketHeader(t *testing.T) {
	size := int(unsafe.Sizeof(PacketHeader))
	if size != 10 {
		t.Fatalf("Unexpected packet size %d", size)
	}
}