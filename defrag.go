package defrag

import (
	"bytes"
	"errors"
	"io"
	"math"
	"net"
	"unsafe"

	"encoding/binary"
)

type payload []byte
type frame struct {
	packets         []payload
	id              uint32
	packetsExpected uint16
	packetsReceived uint16
	size            uint16
}

type chanMessage struct {
	frame *frame
	err   error
}

type PacketConn interface {
    ReadFrom(p []byte) (n int, addr net.Addr, err error)
}

type Defrag struct {
	currentFrameID uint32
	frames map[uint32](*frame)
	connection  PacketConn
	ch chan chanMessage
}

type PacketHeader struct {
	FrameID uint32
	Count   uint16
	Number  uint16
	Length  uint16
}

// Fetch the packet header from a raw packet, return a Go struct
// Cutting corners:
//   * Assume network order
//   * Ignore errors
// Based on https://stackoverflow.com/questions/27814408/working-with-raw-bytes-from-a-network-in-go
func getPacketHeader(data []byte) PacketHeader {
    var packetHeader PacketHeader
    buf := bytes.NewReader(data)
    binary.Read(buf, binary.LittleEndian, &packetHeader)
	return packetHeader
}

func New(connection net.PacketConn) io.Reader {
	return new(connection)
}

func getLimits() (maxPayloadSize int, packetHeaderSize int, maxFrameSize int) {
	maxPayloadSize = math.MaxUint16
	ph := PacketHeader{}
	packetHeaderSize = int(unsafe.Sizeof(ph))
	maxFrameSize = packetHeaderSize + maxPayloadSize
	return 
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
func new(connection PacketConn) io.Reader {
	d := &Defrag {
		frames: make(map[uint32](*frame)),
		connection: connection,
		ch: make(chan chanMessage),
	}

	// Read packets from the connection until an error
	go func(d *Defrag) {
		for {
			_, _, maxFrameSize := getLimits()
			buf := make([]byte, maxFrameSize)
			packetSize, _, err := d.connection.ReadFrom(buf)
			if packetSize > 0 {
				buf = buf[:packetSize]
				d.storeInCache(buf)
			}
			d.flashFullFrames()
			if err != nil {
				d.ch <- chanMessage{err: errors.New("EOF")}
				break
			}	
		}
	}(d)
	return d
}

// Read reads frames from the channel into the provided buffer
// Cutting corners:
//    * User provided buf has enough space for the whole frame?
func (d *Defrag) Read(p []byte) (n int, err error) {
	msg := <- d.ch
	if msg.err != nil {
		return 0, msg.err
	}
	bytesCopied := 0
	frame := msg.frame
	for _, packet := range(frame.packets){
		copy(p[bytesCopied:], []byte(packet))
		bytesCopied += len(packet)
	}
	return bytesCopied, nil
}

func (d *Defrag) flashFullFrames() {
	found := true
	currentFrameID := d.currentFrameID
	for found {
		frameNew, found := d.frames[currentFrameID]
		// I have a complete frame?
		found = found && frameNew.packetsExpected == frameNew.packetsReceived 
		if found {
			delete(d.frames, currentFrameID)
			currentFrameID += 1
			d.ch <- chanMessage{frame:frameNew}
		}
	}

	// Probably a new currentFrameID 
	d.currentFrameID = currentFrameID
}

func (d *Defrag) storeInCache(data []byte) {
	packetHeader := getPacketHeader(data)
	frames := d.frames
	frameNew, found := frames[packetHeader.FrameID]
	if !found {
		frameNew = &frame{
			packets: make([]payload, packetHeader.Count),
			id:      packetHeader.FrameID,

			packetsExpected: packetHeader.Count,
			packetsReceived: 0,
			size:            0,		
		}
	}
	frameNew.packets[packetHeader.Number] = data
	frameNew.packetsReceived += 1
	frameNew.size += uint16(len(data))
	frames[packetHeader.FrameID] = frameNew
}