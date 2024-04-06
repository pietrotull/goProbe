package capture

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/els0r/goProbe/cmd/goProbe/config"
	"github.com/els0r/goProbe/pkg/capture/capturetypes"
	"github.com/els0r/goProbe/pkg/types"
	"github.com/els0r/goProbe/pkg/types/hashmap"
	"github.com/els0r/telemetry/logging"
	"github.com/fako1024/slimcap/capture"
	"github.com/fako1024/slimcap/capture/afpacket/afring"
	"github.com/fako1024/slimcap/link"
)

const (

	// MaxIfaces is the maximum number of interfaces we can monitor
	MaxIfaces = 1024
)

var (

	// ErrLocalBufferOverflow signifies that the local packet buffer is full
	ErrLocalBufferOverflow = errors.New("local packet buffer overflow")

	defaultSourceInitFn = func(c *Capture) (Source, error) {
		return afring.NewSource(c.iface,
			afring.CaptureLength(link.CaptureLengthMinimalIPv6Transport),
			afring.BufferSize(c.config.RingBuffer.BlockSize, c.config.RingBuffer.NumBlocks),
			afring.Promiscuous(c.config.Promisc),
		)
	}
)

// sourceInitFn denotes the function used to initialize a capture source,
// providing the ability to override the default behavior, e.g. in mock tests
type sourceInitFn func(*Capture) (Source, error)

// Captures denotes a named set of Capture instances, wrapping a map and the
// required synchronization of all its actions
type captures struct {
	Map map[string]*Capture
	sync.RWMutex
}

// newCaptures instantiates a new, empty set of Captures
func newCaptures() *captures {
	return &captures{
		Map:     make(map[string]*Capture),
		RWMutex: sync.RWMutex{},
	}
}

// Ifaces return the list of names of all interfaces in the set
func (c *captures) Ifaces(ifaces ...string) []string {
	if len(ifaces) == 0 {
		c.RLock()
		ifaces = make([]string, 0, len(c.Map))
		for iface := range c.Map {
			ifaces = append(ifaces, iface)
		}
		c.RUnlock()
	}

	return ifaces
}

// Get safely returns a Capture by name (and an indicator if it exists)
func (c *captures) Get(iface string) (capture *Capture, exists bool) {
	c.RLock()
	capture, exists = c.Map[iface]
	c.RUnlock()
	return
}

// Set safely adds / overwrites a Capture by name
func (c *captures) Set(iface string, capture *Capture) {
	c.Lock()
	c.Map[iface] = capture
	c.Unlock()
}

// Delete safely removes a Capture from the set by name
func (c *captures) Delete(iface string) {
	c.Lock()
	delete(c.Map, iface)
	c.Unlock()
}

// Capture captures and logs flow data for all traffic on a
// given network interface. For each Capture, a goroutine is
// spawned at creation time. To avoid leaking this goroutine,
// be sure to call Close() when you're done with a Capture.
//
// Each capture is associated with a network interface when created. This interface
// can never be changed.
//
// All public methods of Capture are threadsafe.
type Capture struct {
	iface string

	config config.CaptureConfig

	// stats from the last rotation or reset (needed for Status)
	stats capturetypes.CaptureStats

	// Rotation state synchronization
	capLock *captureLock

	// Logged flows since creation of the capture (note that some
	// flows are retained even after Rotate has been called)
	flowLog *FlowLog

	// Generic handle / source for packet capture
	captureHandle Source
	sourceInitFn  sourceInitFn

	// Error tracking (type / errno specific)
	// parsingErrors ParsingErrTracker

	// WaitGroup tracking active processing
	wgProc sync.WaitGroup

	// startedAt tracks when the capture was started
	startedAt time.Time
}

// newCapture creates a new Capture associated with the given iface.
func newCapture(iface string, config config.CaptureConfig) *Capture {
	return &Capture{
		iface:        iface,
		config:       config,
		capLock:      newCaptureLock(),
		flowLog:      NewFlowLog(),
		sourceInitFn: defaultSourceInitFn,
	}
}

// SetSourceInitFn sets a custom function used to initialize a new capture
func (c *Capture) SetSourceInitFn(fn sourceInitFn) *Capture {
	c.sourceInitFn = fn
	return c
}

// Iface returns the name of the interface
func (c *Capture) Iface() string {
	return c.iface
}

func (c *Capture) run() (err error) {

	// Set up the packet source and capturing
	c.captureHandle, err = c.sourceInitFn(c)
	if err != nil {
		return fmt.Errorf("failed to initialize capture: %w", err)
	}

	// make sure to store when the capture started
	c.startedAt = time.Now()

	return
}

func (c *Capture) close() error {
	if err := c.captureHandle.Close(); err != nil {
		return err
	}

	// Wait until processing has concluded
	c.wgProc.Wait()

	// Setting the handle to nil isn't stricly necessary, but it's an additional
	// guard against races (because it allows the race detector to pick up more
	// easily on potential concurrent accesses) and might trigger a crash on any
	// unwanted access
	c.captureHandle = nil
	return nil
}

func (c *Capture) rotate(ctx context.Context) (agg *hashmap.AggFlowMap) {

	logger := logging.FromContext(ctx)

	// write how many flows are currently in the map
	nFlows := c.flowLog.Len()

	var totals = &types.Counters{}
	defer func() {
		go func(iface string) {
			// write volume metrics to prometheus
			promNumFlows.WithLabelValues(c.iface).Set(float64(nFlows))

			if totals != nil {
				promBytes.WithLabelValues(iface, "inbound").Add(float64(totals.BytesRcvd))
				promBytes.WithLabelValues(iface, "outbound").Add(float64(totals.BytesSent))
				promPackets.WithLabelValues(iface, "inbound").Add(float64(totals.PacketsRcvd))
				promPackets.WithLabelValues(iface, "outbound").Add(float64(totals.PacketsSent))
			}
		}(c.iface)
	}()

	if nFlows == 0 {
		logger.Debug("there are currently no flow records available")
		return
	}
	agg, totals = c.flowLog.Rotate()

	return
}

func (c *Capture) flowMap(ctx context.Context) (agg *hashmap.AggFlowMap) {

	logger := logging.FromContext(ctx)

	if c.flowLog.Len() == 0 {
		logger.Debug("there are currently no flow records available")
		return
	}
	agg = c.flowLog.Aggregate()

	return
}

// process is the heart of the Capture. It listens for network traffic on the
// network interface and logs the corresponding flows.
//
// process keeps running until Close is called on its capture handle or it encounters
// a serious capture error
func (c *Capture) process() <-chan error {

	captureErrors := make(chan error, 64)

	c.wgProc.Add(1)
	go func() {

		defer func() {
			close(captureErrors)
			c.wgProc.Done()
		}()

		// Iniitalize a new local buffer for this interface - this is kept local to avoid
		// any possibility of escaping to the heap and / or accidental misuse of the underlying
		// memory area
		localBuf := new(LocalBuffer)

		// Main packet capture loop which an interface should be in most of the time
		for {

			// Since lock confirmation is only done from a single goroutine (this one)
			// tracking if the capture source was unblocked is safe and can act as flag when to
			// read from the lock request channel (which in turn is atomic).
			// Similarly, once this goroutine observes that the channel length is 1 it is guaranteed
			// that there is a request on the channel that can be read on the next line.
			// This logic may be slightly more contrived than a select{} statement but it increases
			// packet throughput by several percent points
			if len(c.capLock.request) > 0 {
				buf := <-c.capLock.request      // Consume the lock request
				c.capLock.confirm <- struct{}{} // Confirm that process() is not processing

				// Claim / assign the shared data from the memory pool for / to this buffer
				// Release is handled in bufferPackets()
				localBuf.Assign(buf)

				// Continue fetching packets and add them to the local buffer - if the method
				// returns with a non-nil error, it means that graceful termination has been requested
				if err := c.bufferPackets(localBuf, captureErrors); err != nil {
					return
				}

				// Advance to the next loop iteration in case there is a pending lock
				continue
			}

			// Fetch the next packet or PPOLL event from the source
			ipLayer, pktType, pktSize, err := c.captureHandle.NextIPPacketZeroCopy()
			if err != nil {
				if errors.Is(err, capture.ErrCaptureUnblocked) { // capture unblocked

					// Advance to the next loop iteration (during which the pending lock will be
					// consumed / acted on)
					continue
				}
				if errors.Is(err, capture.ErrCaptureStopped) { // capture stopped gracefully
					return
				}

				captureErrors <- fmt.Errorf("capture error: %w", err)
				return
			}
			c.stats.Processed++

			// Parse the packet, extract relevant data and add to the flow log
			// Note: Since the compiler fails to inline this as a function, it is kept in the main loop
			if iplayerType := ipLayer.Type(); iplayerType == ipLayerTypeV4 {
				epHash, direction, errno := ParsePacketV4(ipLayer)
				c.addToFlowLogV4(epHash, pktType, pktSize, direction, errno)
			} else if iplayerType == ipLayerTypeV6 {
				epHash, direction, errno := ParsePacketV6(ipLayer)
				c.addToFlowLogV6(epHash, pktType, pktSize, direction, errno)
			} else {
				c.stats.ParsingErrors[capturetypes.ErrnoInvalidIPHeader]++
			}
		}
	}()

	return captureErrors
}

func (c *Capture) bufferPackets(buf *LocalBuffer, captureErrors chan error) error {

	// Ensure that the buffer is released at the end of the method
	defer buf.Release()

	// Populate the buffer
	for {
		if len(c.capLock.done) > 0 {
			<-c.capLock.done // Consume the unlock request to continue normal processing
			break
		}

		// Fetch the next packet form the wire
		ipLayer, pktType, pktSize, err := c.captureHandle.NextIPPacketZeroCopy()
		if err != nil {

			// If we receive an unblock event while capturing to buffer, continue
			if errors.Is(err, capture.ErrCaptureUnblocked) { // capture unblocked (during lock)
				continue
			}
			if errors.Is(err, capture.ErrCaptureStopped) { // capture stopped gracefully

				// This is the only error we return in order to react with graceful termination
				// in the calling routine
				return err
			}

			captureErrors <- fmt.Errorf("capture error while buffering: %w", err)

			break
		}

		// Parse the packet and extract relevant data for future addition to the flow log
		// Note: Since the compiler fails to inline this as a function, it is kept in the
		// main buffer loop
		if iplayerType := ipLayer.Type(); iplayerType == ipLayerTypeV4 {
			epHash, auxInfo, errno := ParsePacketV4(ipLayer)

			// Try to append to local buffer. In case the buffer is full, stop buffering and
			// wait for the unlock request
			if !buf.Add(epHash[:], pktType, pktSize, true, auxInfo, errno) {
				captureErrors <- ErrLocalBufferOverflow
				<-c.capLock.done // Consume the unlock request to continue normal processing
				break
			}
		} else if iplayerType == ipLayerTypeV6 {
			epHash, auxInfo, errno := ParsePacketV6(ipLayer)

			// Try to append to local buffer. In case the buffer is full, stop buffering and
			// wait for the unlock request
			if !buf.Add(epHash[:], pktType, pktSize, true, auxInfo, errno) {
				captureErrors <- ErrLocalBufferOverflow
				<-c.capLock.done // Consume the unlock request to continue normal processing
				break
			}
		} else {
			c.stats.ParsingErrors[capturetypes.ErrnoInvalidIPHeader]++
		}
	}

	// Drain the buffer (if not empty)
	for {
		epHash, pktType, pktSize, isIPv4, auxInfo, errno, ok := buf.Next()
		if !ok {
			break
		}
		c.stats.Processed++

		if isIPv4 {
			c.addToFlowLogV4(capturetypes.EPHashV4(epHash), pktType, pktSize, auxInfo, errno)
			continue
		}
		c.addToFlowLogV6(capturetypes.EPHashV6(epHash), pktType, pktSize, auxInfo, errno)
	}

	// Update the buffer usage gauge for this interface and release the buffer
	promGlobalBufferUsage.WithLabelValues(c.iface).Set(buf.Usage())

	return nil
}

func (c *Capture) addToFlowLogV4(epHash capturetypes.EPHashV4, pktType byte, pktSize uint32, auxInfo byte, errno capturetypes.ParsingErrno) {

	// Parse / add the received data to the map of flows
	if errno > capturetypes.ErrnoOK {
		if errno.ParsingFailed() {
			c.stats.ParsingErrors[errno]++
		}
		return
	}

	// Predict if the packet is most likely to trigger the reverse hash lookup and start with that flow then
	if epHash.IsProbablyReverse() {
		epHashReverse := epHash.Reverse()
		if flowToUpdate, existsReverseHash := c.flowLog.flowMapV4[string(epHashReverse[:])]; existsReverseHash {
			flowToUpdate.UpdateFlow(pktType, pktSize)
		} else if flowToUpdate, existsHash := c.flowLog.flowMapV4[string(epHash[:])]; existsHash {
			flowToUpdate.UpdateFlow(pktType, pktSize)
		} else {
			if direction := capturetypes.ClassifyPacketDirectionV4(epHash, auxInfo); direction == capturetypes.DirectionReverts {
				c.flowLog.flowMapV4[string(epHashReverse[:])] = NewFlow(pktType, pktSize)
			} else {
				c.flowLog.flowMapV4[string(epHash[:])] = NewFlow(pktType, pktSize)
			}
		}
		return
	}

	// Update or assign the flow in forward lookup mode first
	if flowToUpdate, existsHash := c.flowLog.flowMapV4[string(epHash[:])]; existsHash {
		flowToUpdate.UpdateFlow(pktType, pktSize)
	} else {
		epHashReverse := epHash.Reverse()
		if flowToUpdate, existsReverseHash := c.flowLog.flowMapV4[string(epHashReverse[:])]; existsReverseHash {
			flowToUpdate.UpdateFlow(pktType, pktSize)
		} else {
			if direction := capturetypes.ClassifyPacketDirectionV4(epHash, auxInfo); direction == capturetypes.DirectionReverts {
				c.flowLog.flowMapV4[string(epHashReverse[:])] = NewFlow(pktType, pktSize)
			} else {
				c.flowLog.flowMapV4[string(epHash[:])] = NewFlow(pktType, pktSize)
			}
		}
	}
}

func (c *Capture) addToFlowLogV6(epHash capturetypes.EPHashV6, pktType byte, pktSize uint32, auxInfo byte, errno capturetypes.ParsingErrno) {

	// Parse / add the received data to the map of flows
	if errno > capturetypes.ErrnoOK {
		if errno.ParsingFailed() {
			c.stats.ParsingErrors[errno]++
		}
		return
	}

	// Predict if the packet is most likely to trigger the reverse hash lookup and start with that flow then
	if epHash.IsProbablyReverse() {
		epHashReverse := epHash.Reverse()
		if flowToUpdate, existsReverseHash := c.flowLog.flowMapV6[string(epHashReverse[:])]; existsReverseHash {
			flowToUpdate.UpdateFlow(pktType, pktSize)
		} else if flowToUpdate, existsHash := c.flowLog.flowMapV6[string(epHash[:])]; existsHash {
			flowToUpdate.UpdateFlow(pktType, pktSize)
		} else {
			if direction := capturetypes.ClassifyPacketDirectionV6(epHash, auxInfo); direction == capturetypes.DirectionReverts {
				c.flowLog.flowMapV6[string(epHashReverse[:])] = NewFlow(pktType, pktSize)
			} else {
				c.flowLog.flowMapV6[string(epHash[:])] = NewFlow(pktType, pktSize)
			}
		}
		return
	}

	// Update or assign the flow in forward lookup mode first
	if flowToUpdate, existsHash := c.flowLog.flowMapV6[string(epHash[:])]; existsHash {
		flowToUpdate.UpdateFlow(pktType, pktSize)
	} else {
		epHashReverse := epHash.Reverse()
		if flowToUpdate, existsReverseHash := c.flowLog.flowMapV6[string(epHashReverse[:])]; existsReverseHash {
			flowToUpdate.UpdateFlow(pktType, pktSize)
		} else {
			if direction := capturetypes.ClassifyPacketDirectionV6(epHash, auxInfo); direction == capturetypes.DirectionReverts {
				c.flowLog.flowMapV6[string(epHashReverse[:])] = NewFlow(pktType, pktSize)
			} else {
				c.flowLog.flowMapV6[string(epHash[:])] = NewFlow(pktType, pktSize)
			}
		}
	}
}

func (c *Capture) status() (*capturetypes.CaptureStats, error) {

	stats, err := c.captureHandle.Stats()
	if err != nil {
		return nil, err
	}

	c.stats.ReceivedTotal += stats.PacketsReceived
	c.stats.ProcessedTotal += c.stats.Processed
	c.stats.DroppedTotal += stats.PacketsDropped

	// add exposed metrics
	// we do this every 5 minutes only in order not to interfere with the
	// main packet processing loop. If this counter moves slowly (as in gets
	// gets an update only every 5 minutes) it's not an issue to understand
	// processed data volumes across longer time frames
	go func(iface string, processed, dropped, errors uint64) {
		promPacketsProcessed.WithLabelValues(iface).Add(float64(processed))
		promPacketsDropped.WithLabelValues(iface).Add(float64(dropped))
		promCaptureErrors.WithLabelValues(iface).Add(float64(errors))
	}(c.iface, c.stats.Processed, stats.PacketsDropped, uint64(c.stats.ParsingErrors.Sum()))

	res := capturetypes.CaptureStats{
		StartedAt:      c.startedAt,
		Received:       stats.PacketsReceived,
		ReceivedTotal:  c.stats.ReceivedTotal,
		Processed:      c.stats.Processed,
		ProcessedTotal: c.stats.ProcessedTotal,
		Dropped:        stats.PacketsDropped,
		DroppedTotal:   c.stats.DroppedTotal,
		ParsingErrors:  c.stats.ParsingErrors,
	}

	c.stats.Processed = 0
	c.stats.ParsingErrors.Reset()

	return &res, nil
}

func (c *Capture) fetchStatusInBackground(ctx context.Context) (res chan *capturetypes.CaptureStats) {
	res = make(chan *capturetypes.CaptureStats)

	// Extract capture stats in a separate goroutine to minimize time-to-unblock
	// This should be finished by the time the rotation has taken place (at which
	// time the stats can be pulled from the returned channel)
	go func() {
		stats, err := c.status()
		if err != nil {
			logging.FromContext(ctx).Errorf("failed to get capture stats: %v", err)
		}

		res <- stats
		close(res)
	}()

	return
}

func (c *Capture) lock() {

	// Fetch data from the pool for the local buffer. Tis will wait until it is actually
	// available, allowing us to use a single buffer for all interfaces
	buf := memPool.Get(0)

	// Notify the capture that a locked interaction is about to begin, then
	// unblock the capture potentially being in a blocking PPOLL syscall
	// Channel has a depth of one and hence this push is non-blocking. Since
	// we wait for confirmation there is no possibility of repeated attempts
	// or race conditions
	c.capLock.request <- buf
	if err := c.captureHandle.Unblock(); err != nil {
		panic(fmt.Sprintf("unexpectedly failed to unblock capture handle, deadlock inevitable: %s", err))
	}

	// Wait for confirmation of reception from the processing routine
	<-c.capLock.confirm
}

func (c *Capture) unlock() {

	// Signal that the rotation is complete, releasing the processing routine
	// Since the done channel has a depth of one an Unblock() event needs to be
	// sent to ensure that a capture currently waiting for packets in the buffering
	// state continues to the next iteration in order to observe the unlock request
	c.capLock.done <- struct{}{}
	if err := c.captureHandle.Unblock(); err != nil {
		panic(fmt.Sprintf("unexpectedly failed to unblock capture handle, deadlock inevitable: %s", err))
	}
}

type captureLock struct {
	request chan []byte
	confirm chan struct{}
	done    chan struct{}
}

func newCaptureLock() *captureLock {
	return &captureLock{
		request: make(chan []byte, 1),
		confirm: make(chan struct{}),
		done:    make(chan struct{}, 1),
	}
}
