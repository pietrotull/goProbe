package capture

import (
	"fmt"
	"unsafe"

	"github.com/els0r/goProbe/cmd/goProbe/config"
	"github.com/els0r/goProbe/pkg/capture/capturetypes"
	"github.com/fako1024/gotools/concurrency"
	"golang.org/x/sys/unix"
)

const (

	// bufElementAddSize denotes the required (additional) size for a buffer element
	// (size of EPHash + 4 bytes for pktSize + 1 byte for pktType, isIPv4, errno, respectively)
	bufElementAddSize = 7
)

var (

	// Initial size of a buffer
	initialBufferSize = unix.Getpagesize()

	// Global (limited) memory pool used to minimize allocations
	memPool       = concurrency.NewMemPool(config.DefaultLocalBufferNumBuffers)
	maxBufferSize = config.DefaultLocalBufferSizeLimit
)

// LocalBuffer denotes a local packet buffer used to temporarily capture packets
// from the source (e.g. during rotation) to avoid a ring / kernel buffer overflow
type LocalBuffer struct {
	data        []byte // continuous buffer slice
	writeBufPos int    // current position in buffer slice
	readBufPos  int    // current position in buffer slice
}

// Assign sets the actual underlying data slice (obtained from a memory pool) of this buffer
func (l *LocalBuffer) Assign(data []byte) {
	l.data = data

	// Ascertain the current size of the underlying data slice and grow if required
	if len(l.data) == 0 {
		l.data = make([]byte, initialBufferSize)
	}
}

// Release returns the data slice to the memory pool and resets the buffer position
func (l *LocalBuffer) Release() {
	memPool.Put(l.data)
	l.writeBufPos = 0
	l.readBufPos = 0
	l.data = nil
}

// Usage return the relative fraaction of the buffer capacity in use (i.e. written to, independent of
// number of items already retreived by Next())
func (l *LocalBuffer) Usage() float64 {

	// Note: maxBufferSize is guarded against zero in setLocalBuffers(), so this cannot cause division by zero
	return float64(l.writeBufPos) / float64(maxBufferSize)
}

// Add adds an element to the buffer, returning ok = true if successful
// If the buffer is full / may not grow any further, ok is false
func (l *LocalBuffer) Add(epHash []byte, pktType byte, pktSize uint32, isIPv4 bool, auxInfo byte, errno capturetypes.ParsingErrno) (ok bool) {

	// If required, attempt to grow the buffer
	if l.writeBufPos+len(epHash)+bufElementAddSize >= len(l.data) {

		// If the buffer size is already at its limit, reject the new element
		if len(l.data) >= maxBufferSize {
			return false
		}

		l.grow(min(maxBufferSize, 2*len(l.data)))
	}

	// Transfer data to the buffer
	if isIPv4 {
		l.data[l.writeBufPos] = 0
		copy(l.data[l.writeBufPos+1:l.writeBufPos+capturetypes.EPHashSizeV4+1], epHash)

		l.data[l.writeBufPos+capturetypes.EPHashSizeV4+1] = pktType
		l.data[l.writeBufPos+capturetypes.EPHashSizeV4+2] = auxInfo
		*(*int8)(unsafe.Pointer(&l.data[l.writeBufPos+capturetypes.EPHashSizeV4+3])) = int8(errno) // #nosec G103
		*(*uint32)(unsafe.Pointer(&l.data[l.writeBufPos+capturetypes.EPHashSizeV4+4])) = pktSize   // #nosec G103

		// Increment buffer position
		l.writeBufPos += capturetypes.EPHashSizeV4 + bufElementAddSize

		return true
	}

	l.data[l.writeBufPos] = 1
	copy(l.data[l.writeBufPos+1:l.writeBufPos+capturetypes.EPHashSizeV6+1], epHash)

	l.data[l.writeBufPos+capturetypes.EPHashSizeV6+1] = pktType
	l.data[l.writeBufPos+capturetypes.EPHashSizeV6+2] = auxInfo
	*(*int8)(unsafe.Pointer(&l.data[l.writeBufPos+capturetypes.EPHashSizeV6+3])) = int8(errno) // #nosec G103
	*(*uint32)(unsafe.Pointer(&l.data[l.writeBufPos+capturetypes.EPHashSizeV6+4])) = pktSize   // #nosec G103

	// Increment buffer position
	l.writeBufPos += capturetypes.EPHashSizeV6 + bufElementAddSize

	return true
}

// Next fetches the i-th element from the buffer
func (l *LocalBuffer) Next() ([]byte, byte, uint32, bool, byte, capturetypes.ParsingErrno, bool) {

	if l.readBufPos >= l.writeBufPos {
		return nil, 0, 0, false, 0, 0, false
	}

	pos := l.readBufPos
	if l.data[pos] == 0 {
		l.readBufPos += capturetypes.EPHashSizeV4 + bufElementAddSize
		return l.data[pos+1 : pos+1+capturetypes.EPHashSizeV4],
			l.data[pos+1+capturetypes.EPHashSizeV4],
			*(*uint32)(unsafe.Pointer(&l.data[pos+capturetypes.EPHashSizeV4+4])), // #nosec G103
			true,
			l.data[pos+capturetypes.EPHashSizeV4+2],
			capturetypes.ParsingErrno(*(*int8)(unsafe.Pointer(&l.data[pos+capturetypes.EPHashSizeV4+3]))), // #nosec G103
			true
	}

	l.readBufPos += capturetypes.EPHashSizeV6 + bufElementAddSize
	return l.data[pos+1 : pos+1+capturetypes.EPHashSizeV6],
		l.data[pos+1+capturetypes.EPHashSizeV6],
		*(*uint32)(unsafe.Pointer(&l.data[pos+capturetypes.EPHashSizeV6+4])), // #nosec G103
		false,
		l.data[pos+capturetypes.EPHashSizeV6+2],
		capturetypes.ParsingErrno(*(*int8)(unsafe.Pointer(&l.data[pos+capturetypes.EPHashSizeV6+3]))), // #nosec G103
		true
}

///////////////////////////////////////////////////////////////////////////////////

// setLocalBuffers sets the number of (and hence the maximum concurrency for Status() calls) and
// maximum size of the local memory buffers (globally, not per interface)
func setLocalBuffers(nBuffers, sizeLimit int) error {

	// Guard against invalid (i.e. zero) buffer size / limits
	if nBuffers == 0 || sizeLimit == 0 {
		return fmt.Errorf("invalid number of local buffers (%d) / size limit (%d) specified", nBuffers, sizeLimit)
	}

	if memPool != nil {
		memPool.Clear()
	}
	memPool = concurrency.NewMemPool(nBuffers)
	maxBufferSize = sizeLimit

	return nil
}

func (l *LocalBuffer) grow(newSize int) {
	newData := make([]byte, newSize)
	copy(newData, l.data)
	l.data = newData
}
