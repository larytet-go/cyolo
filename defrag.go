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

	// I keep frames in a map
	frame struct {
		packets []payload
		id      uint32
		missing uint16
		size    uint32
	}

	// An API's user blocks on read from this channel
	// until a whole frame collected
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

	// An essence of net.PacketConn interface
	// I need a simpler interface for unitests
	PacketConn interface {
		ReadFrom(p []byte) (n int, addr net.Addr, err error)
	}

	// Sate of the defagmentator
	State struct {
		currentFrameID uint32

		// I keep incoming packets (fragments) here
		frames     map[uint32](*frame)
		connection PacketConn
		ch         chan chanMessage
		err        error
	}
)

const (
	maxPayloadSize = math.MaxUint16

	// Unfortunately PacketHeader is padded
	// ph := PacketHeader{}
	// packetHeaderSize = int(unsafe.Sizeof(ph)) gives 12
	packetHeaderSize = 10
	maxFrameSize     = packetHeaderSize + maxPayloadSize
)

// Create a new Defragmentation API
func New(connection net.PacketConn) io.Reader {
	return new(connection)
}

// Blocking Read
// Read reads a whole frame from the channel
// Copies the data from the frame into the provided by the user buffer
// Cutting corners:
//    * Provided by the user 'buf' has enough space for the whole frame
//    * Parallel Calls to Read() after EOF can block
func (s *State) Read(p []byte) (n int, err error) {
	if s.err != nil {
		return 0, s.err
	}
	msg := <-s.ch
	s.err = msg.err // A race condition here!
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

// Fetch the packet header from a raw packet
// Network order?
func (ph *PacketHeader) read(data []byte) {
	ph.FrameID = binary.BigEndian.Uint32(data[0:])
	ph.Count = binary.BigEndian.Uint16(data[4:])
	ph.Number = binary.BigEndian.Uint16(data[6:])
	ph.Length = binary.BigEndian.Uint16(data[8:])
}

// Setup a packet header in a raw packet
// Network order?
func (ph *PacketHeader) write(data []byte) {
	binary.BigEndian.PutUint32(data[0:], ph.FrameID)
	binary.BigEndian.PutUint16(data[4:], ph.Count)
	binary.BigEndian.PutUint16(data[6:], ph.Number)
	binary.BigEndian.PutUint16(data[8:], ph.Length)
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
	d := &State{
		frames:     make(map[uint32](*frame)),
		connection: connection,
		ch:         make(chan chanMessage),
	}

	// Read packets from the connection until an error
	// One thread does it all, no need for synchronization
	go func(s *State) {
		for {
			buf := make([]byte, maxFrameSize)
			packetSize, _, err := s.connection.ReadFrom(buf)
			if packetSize > 0 {
				// Assume that ReadFrom returns the whole packet
				buf = buf[:packetSize]
				s.storeInCache(buf)
			}
			// I can call flashFullFrames() only if the frame ID == s.currentFrameID
			// and save a few lookups in the map
			s.flashFullFrames()
			if err != nil {
				s.ch <- chanMessage{err: errors.New("EOF")}
				break
			}
		}
	}(d)
	return d
}

// Check if currentFrameID is in the cache and completed
// If I have a whole frame send the frame to the client
// increment the currentFrameID
func (s *State) flashFullFrames() {
	found := true
	currentFrameID := s.currentFrameID
	frame := &frame{}
	for found {
		frame, found = s.frames[currentFrameID]
		// I have a complete frame?
		found = found && (frame.missing == 0)
		if found {
			s.ch <- chanMessage{frame: frame}
			delete(s.frames, currentFrameID)
			currentFrameID += 1
		}
	}

	// Probably a new currentFrameID
	s.currentFrameID = currentFrameID
}

// Fetch the packet header
// If cache miss add a new frame to the cache
// If cache hit update the frame in the cache
// Cutting corners:
//  * I do not expect duplicate packets
//  * RAM is unlimited
//  * 'map' never overflows
//  * I do not check packetHeader.Length (payload length)
func (s *State) storeInCache(data []byte) {
	packetHeader := &PacketHeader{}
	packetHeader.read(data)
	frames := s.frames
	cachedFrame, found := frames[packetHeader.FrameID]
	if !found {
		cachedFrame = &frame{
			packets: make([]payload, packetHeader.Count),
			id:      packetHeader.FrameID,
			missing: packetHeader.Count,
		}
	}
	payload := data[packetHeaderSize:]
	cachedFrame.packets[packetHeader.Number] = payload
	cachedFrame.missing -= 1
	cachedFrame.size += uint32(len(payload))
	frames[packetHeader.FrameID] = cachedFrame
}
