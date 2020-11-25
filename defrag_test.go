package defrag

import (
	"bytes"
	"errors"
	"net"
	"testing"

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
		FrameID: c.frame,
		Count:   c.packet,
		Number:  3,
		Length:  1,
	}
	// https://stackoverflow.com/questions/27814408/working-with-raw-bytes-from-a-network-in-go
	buf := bytes.NewBuffer(p)
	binary.Write(buf, binary.LittleEndian, packetHeader)
	_, packetHeaderSize, _ := getLimits()

	p[packetHeaderSize] = uint8(c.packet)
	c.packet += 1
	return (packetHeaderSize+1), nil, nil

}

func Test_Read(t *testing.T) {
	packetConnMock := &PacketConnMock {
		packets: 3,
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


// Check padding of the PacketHeader
func Test_PacketHeader(t *testing.T) {
	_, packetHeaderSize, _ := getLimits()
	if packetHeaderSize != 10 {
		t.Fatalf("Unexpected packet size %d", packetHeaderSize)
	}
}