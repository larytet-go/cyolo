package defrag

import (
	"errors"
	"io"
	"math"
	"net"

	"encoding/binary"
)

type (
	payload []byte
	frame   struct {
		packets []payload
		id      uint32
		missing uint16
		size    uint16
	}

	chanMessage struct {
		frame *frame
		err   error
	}

	PacketHeader struct {
		FrameID uint32
		Count   uint16
		Number  uint16
		Length  uint16
	}

	PacketConn interface {
		ReadFrom(p []byte) (n int, addr net.Addr, err error)
	}

	Defrag struct {
		currentFrameID uint32
		frames         map[uint32](*frame)
		connection     PacketConn
		ch             chan chanMessage
	}
)

const (
	maxPayloadSize = math.MaxUint16

	// Unfortunately PacketHeader is padded
	// packetHeaderSize = int(unsafe.Sizeof(ph)) is 12
	// ph := PacketHeader{}

	packetHeaderSize = 10 //
	maxFrameSize     = packetHeaderSize + maxPayloadSize
)

// Create a new Defragmentation API
func New(connection net.PacketConn) io.Reader {
	return new(connection)
}

// Blocking Read
// Read reads frames from the channel into the provided by the user buffer
// Cutting corners:
//    * Provided by the user 'buf' has enough space for the whole frame
func (d *Defrag) Read(p []byte) (n int, err error) {
	msg := <-d.ch
	if msg.err != nil {
		return 0, msg.err
	}
	bytesCopied := 0
	frame := msg.frame
	for _, packet := range frame.packets {
		copy(p[bytesCopied:], []byte(packet))
		bytesCopied += len(packet)
	}
	return bytesCopied, nil
}

// Fetch the packet header from a raw packet, return a Go struct
// Network order?
func getPacketHeader(data []byte) PacketHeader {
	packetHeader := PacketHeader{
		FrameID: binary.BigEndian.Uint32(data[0:]),
		Count:   binary.BigEndian.Uint16(data[4:]),
		Number:  binary.BigEndian.Uint16(data[6:]),
		Length:  binary.BigEndian.Uint16(data[8:]),
	}
	return packetHeader
}

// Setup a packet header in a raw packet
// Network order?
func setPacketHeader(data []byte, packetHeader PacketHeader) {
	binary.BigEndian.PutUint32(data[0:], packetHeader.FrameID)
	binary.BigEndian.PutUint16(data[4:], packetHeader.Count)
	binary.BigEndian.PutUint16(data[6:], packetHeader.Number)
	binary.BigEndian.PutUint16(data[8:], packetHeader.Length)
}

// Defrag reads fragments of the packets from the connection
// collects packets in a cache. When all packets of a frame are collected writes
// the whole frame to the output channel
// Cutting corners:
//  * Assume that all fragments arrive, no timeout
//  * No error checks
//  * The RAM is unlimited
//  * Assume wrap around of the 32 bits unsigned frame ID
//  * No initial synchronization: first frame has ID 0
//  * I read a whole packet every time
func new(connection PacketConn) io.Reader {
	d := &Defrag{
		frames:     make(map[uint32](*frame)),
		connection: connection,
		ch:         make(chan chanMessage),
	}

	// Read packets from the connection until an error
	// One thread does it all, no need for synchronization
	go func(d *Defrag) {
		for {
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

// Check if currentFrameID is in the cache and completed
// If I have a whole frame send the frame to the client
// increment the currentFrameID
func (d *Defrag) flashFullFrames() {
	found := true
	currentFrameID := d.currentFrameID
	frame := &frame{}
	for found {
		frame, found = d.frames[currentFrameID]
		// I have a complete frame?
		found = found && (frame.missing == 0)
		if found {
			d.ch <- chanMessage{frame: frame}
			delete(d.frames, currentFrameID)
			currentFrameID += 1
		}
	}

	// Probably a new currentFrameID
	d.currentFrameID = currentFrameID
}

// Fetch the packet header
// If cache miss add add a new frame to the cache
// If cache hit update the frame in the cache
func (d *Defrag) storeInCache(data []byte) {
	packetHeader := getPacketHeader(data)
	frames := d.frames
	cachedFrame, found := frames[packetHeader.FrameID]
	if !found {
		cachedFrame = &frame{
			packets: make([]payload, packetHeader.Count),
			id:      packetHeader.FrameID,

			missing: packetHeader.Count,
			size:    0,
		}
	}
	payload := data[packetHeaderSize:]
	cachedFrame.packets[packetHeader.Number] = payload
	cachedFrame.missing -= 1
	cachedFrame.size += uint16(len(data))
	frames[packetHeader.FrameID] = cachedFrame
}
