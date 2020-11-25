package defrag

import (
	"io"
	"math"
	"net"

	gocache "github.com/patrickmn/go-cache"
)

type frame struct {
	packets []([]byte)
	id   uint32
	packetsExpected uint16
	packetsReceived uint16
	payloadLen uint16

}
const(
	FrameIDSize = 4 
	TotalPacketsSize = 2
	PacketNumberSize = 2
	PayloadLengthSize = 2
	PayloadSize = math.MaxUint16
	MaxFrameSize = FrameIDSize + TotalPacketsSize + PacketNumberSize + PayloadLengthSize + PayloadSize
) 

type Defrag struct {
	lastFrameID uint32
	frames   gocache.Cache
	connection  net.PacketConn
	c chan frame
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
func New(func(connection net.PacketConn) io.Reader {
	d := &Defrag {
		frames:  gocache.New(gocache.NoExpiration, gocache.NoExpiration),
		connection: connection,
		c: make(chan frame)
	}
	go func(d *Defrag) {
		buf := make([]byte, 
		n int, addr Addr, err error := d.connection.ReadFrom()
	}

	return d
}

// Read reads frames from the channel into the provided buffer
// Cutting corners:
//    * ??
func (d *Defrag) Read(p []byte) (n int, err error) {

}