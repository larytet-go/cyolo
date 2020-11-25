package defrag

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"

	"encoding/binary"
)

// Check padding of the PacketHeader
func Test_PacketHeader(t *testing.T) {
	_, packetHeaderSize, _ := getLimits()
	if packetHeaderSize != 10 {
		t.Fatalf("Unexpected packet size %d", packetHeaderSize)
	}
}

type PacketConnMock struct {
	frame   uint32
	packet  uint16
	packets uint16
	t       *testing.T
}

func (c *PacketConnMock) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	t := c.t
	if c.packet > c.packets {
		return 0, nil, errors.New("EOF")
	}

	packetHeader := PacketHeader {
		FrameID: c.frame,
		Count:   c.packets,
		Number:  c.packet,
		Length:  1,
	}
	fmt.Printf("test packetHeader=%v\n", packetHeader)
	setPacketHeader(p, packetHeader)
	p[packetHeaderSize] = uint8(c.packet)
	c.packet += 1
	fmt.Printf("p=%v\n", p[:packetHeaderSize+1])
	return (packetHeaderSize+1), nil, nil

}

func Test_Read(t *testing.T) {
	packetConnMock := &PacketConnMock {
		packets: 3,

		t: t,
	}
	reader := new(packetConnMock)
	buf := make([]byte, 1024)
	count, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if count != 3 {
		t.Fatalf("Unexpected frame size %d", count)
	}
	_, err = reader.Read(buf)
	if err == nil {
		t.Fatalf("Expected error")
	}
}
