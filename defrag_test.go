package defrag

import (
	"errors"
	"net"
	"testing"
)

// Check padding of the PacketHeader
func Test_PacketHeader(t *testing.T) {
	if packetHeaderSize != 10 {
		t.Fatalf("Unexpected packet size %d", packetHeaderSize)
	}
}

type PacketConnMockPacket struct {
	frame   uint32
	packet  uint16
	packets uint16
}

type PacketConnMock struct {
	packets   []PacketConnMockPacket
	packetIdx int
	t         *testing.T
}

func (c *PacketConnMock) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if c.packetIdx >= len(c.packets) {
		return 0, nil, errors.New("EOF")
	}

	packet := c.packets[c.packetIdx]
	c.packetIdx += 1
	packetHeader := PacketHeader{
		FrameID: packet.frame,
		Count:   packet.packets,
		Number:  packet.packet,
		Length:  1,
	}
	packetHeader.write(p)
	p[packetHeaderSize] = uint8(packet.frame)
	return (packetHeaderSize + 1), nil, nil
}

// Cutting corners:
//  * Only basics
func Test_Read(t *testing.T) {
	packetConnMock := &PacketConnMock{
		packets: []PacketConnMockPacket{
			PacketConnMockPacket{1, 1, 2},
			PacketConnMockPacket{0, 1, 2},
			PacketConnMockPacket{1, 0, 2},
			PacketConnMockPacket{0, 0, 2},
		},
		t: t,
	}
	reader := new(packetConnMock)
	buf := make([]byte, 1024)
	for i := 0; i < 2; i++ {
		count, err := reader.Read(buf)
		if err != nil {
			t.Fatalf("Unexpected error %v", err)
		}
		if count != 2 {
			t.Fatalf("Unexpected frame size %d", count)
		}
		packetIdx := int(buf[0])
		t.Logf("i = %d, packetIdx=%d", i, packetIdx)
		if packetIdx != i {
			t.Fatalf("Unexpected index %d", packetIdx)
		}
	}
	_, err := reader.Read(buf)
	if err == nil {
		t.Fatalf("Expected error")
	}
}
