package idgen

import (
	"errors"
	"sync"
	"time"
)

const (
	epoch        int64 = 1735689600000 // custom epoch, ms since Unix epoch
	nodeIDBits   uint8 = 10
	sequenceBits uint8 = 12

	maxNodeID   int64 = -1 ^ (-1 << nodeIDBits)
	maxSequence int64 = -1 ^ (-1 << sequenceBits)
	nodeIDShift uint8 = sequenceBits
	timeShift   uint8 = sequenceBits + nodeIDBits
)

type Generator struct {
	mu            sync.Mutex
	nodeID        int64
	lastTimestamp int64
	sequence      int64
}

func NewGenerator(nodeID int64) (*Generator, error) {
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, errors.New("idgen: node ID out of range")
	}

	return &Generator{
		nodeID: nodeID,
	}, nil

}

func (g *Generator) NextID() (uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli() - epoch

	// Clock has moved backwards.
	if now < g.lastTimestamp {
		return 0, errors.New("idgen: system clock moved backwards")
	}

	// Same millisecond: increment sequence.
	if now == g.lastTimestamp {
		g.sequence++

		// Sequence exhausted: wait for the next millisecond.
		if g.sequence > maxSequence {
			for now <= g.lastTimestamp {
				now = time.Now().UnixMilli() - epoch
			}

			g.sequence = 0
		}
	} else {
		// New millisecond.
		g.sequence = 0
	}

	// Update last timestamp.
	g.lastTimestamp = now

	// Assemble the ID:
	// [ timestamp ][ node ID ][ sequence ]
	id := (uint64(now) << timeShift) |
		(uint64(g.nodeID) << nodeIDShift) |
		uint64(g.sequence)

	return id, nil

}
