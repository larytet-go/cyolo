package defrag

import (
	"bytes"
	"io"
	"math"
	"net"
	"unsafe"

	"encoding/binary"

	gocache "github.com/patrickmn/go-cache"
)

type frame struct {
	packets         []([]byte)
	id              uint32
	packetsExpected uint16
	packetsReceived uint16
	payloadLen      uint16

}

type Defrag struct {
	currentFrameID uint32
	frames   gocache.Cache
	connection  net.PacketConn
	c chan frame
}

type PacketHeader {
	frameID uint32
	packets uint16
	number  uint16
	length  uint16
}

const(
	PayloadSize = math.MaxUint16
	MaxFrameSize = unsafe.Sizeof(PacketHeader) + PayloadSize
) 

// Fetch the packet header from a raw packet, return a Go struct
// Cutting corners:
//   * Assume network order
//   * Ignore errors
// Based on https://stackoverflow.com/questions/27814408/working-with-raw-bytes-from-a-network-in-go
func getPacketHeader(data []byte) packetHeader {
    var packetHeader PacketHeader
    buf := bytes.NewReader(data)
    _ = binary.Read(buf, binary.LittleEndian, &packetHeader)
	return packetHeader
}

// Defrag reads fragments of the packets from the connection 
// collects packets in a cache. When all packets of a frame are collected writes
// the whole frame to the output channel
// Cutting corners: 
//  * Assume that all fragments arrive, no timeout 
//  * No error checks 
//  * The RAM is unlimited 
//  * Assume wrap around of the the 32 bits unsigned frame ID
//  * No initial synchronization: first frame has ID 0 
//  * I read a whole packet every time
func New(func(connection net.PacketConn) io.Reader {
	d := &Defrag {
		frames:  gocache.New(gocache.NoExpiration, gocache.NoExpiration),
		connection: connection,
		c: make(chan frame)
	}
	go func(d *Defrag) {
		buf := make([]byte, MaxFrameSize)
		packetSize , _, err := d.connection.ReadFrom()
		if n > 0 {
			buf = buf[:packetSize]
			d.storeInCache(buf)
		}
		d.flashFullFrames()
	}

	return d
}

// Read reads frames from the channel into the provided buffer
// Cutting corners:
//    * User provided buf has enough space for the whole frame
func (d *Defrag) Read(p []byte) (n int, err error) {
	frame <- d.c

}

func (d *Defrag) storeInCache(data []byte) {
	packetHeader := getPacketHeader(buf)
}