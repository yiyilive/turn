package allocation

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/logging"
	"github.com/pion/transport/vnet"
)

// ManagerConfig a bag of config params for Manager.
type ManagerConfig struct {
	LeveledLogger         logging.LeveledLogger
	Net                   *vnet.Net
	RelayAddressGenerator func() (net.PacketConn, net.Addr, error)
}

// Manager is used to hold active allocations
type Manager struct {
	lock                  sync.RWMutex
	allocations           map[string]*Allocation
	log                   logging.LeveledLogger
	net                   *vnet.Net
	relayAddressGenerator func() (net.PacketConn, net.Addr, error)
}

// NewManager creates a new instance of Manager.
func NewManager(config ManagerConfig) (*Manager, error) {
	if config.Net == nil {
		config.Net = vnet.NewNet(nil) // defaults to native operation
	}

	if config.RelayAddressGenerator == nil {
		return nil, fmt.Errorf("RelayAddressGenerator must be set")
	} else if config.LeveledLogger == nil {
		return nil, fmt.Errorf("LeveledLogger must be set")
	}

	return &Manager{
		log:                   config.LeveledLogger,
		net:                   config.Net,
		allocations:           make(map[string]*Allocation, 64),
		relayAddressGenerator: config.RelayAddressGenerator,
	}, nil
}

// GetAllocation fetches the allocation matching the passed FiveTuple
func (m *Manager) GetAllocation(fiveTuple *FiveTuple) *Allocation {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.allocations[fiveTuple.Fingerprint()]
}

// Close closes the manager and closes all allocations it manages
func (m *Manager) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, a := range m.allocations {
		if err := a.Close(); err != nil {
			return err
		}

	}
	return nil
}

// CreateAllocation creates a new allocation and starts relaying
func (m *Manager) CreateAllocation(
	fiveTuple *FiveTuple,
	turnSocket net.PacketConn,
	requestedPort int,
	lifetime time.Duration) (*Allocation, error) {

	switch {
	case fiveTuple == nil:
		return nil, fmt.Errorf("Allocations must not be created with nil FivTuple")
	case fiveTuple.SrcAddr == nil:
		return nil, fmt.Errorf("Allocations must not be created with nil FiveTuple.SrcAddr")
	case fiveTuple.DstAddr == nil:
		return nil, fmt.Errorf("Allocations must not be created with nil FiveTuple.DstAddr")
	case turnSocket == nil:
		return nil, fmt.Errorf("Allocations must not be created with nil turnSocket")
	case lifetime == 0:
		return nil, fmt.Errorf("Allocations must not be created with a lifetime of 0")
	}

	if a := m.GetAllocation(fiveTuple); a != nil {
		return nil, fmt.Errorf("Allocation attempt created with duplicate FiveTuple %v", fiveTuple)
	}
	a := NewAllocation(turnSocket, fiveTuple, m.log)

	conn, relayAddr, err := m.relayAddressGenerator()
	if err != nil {
		return nil, err
	}

	a.RelaySocket = conn
	a.RelayAddr = relayAddr

	m.log.Debugf("listening on relay addr: %s", a.RelayAddr.String())

	a.lifetimeTimer = time.AfterFunc(lifetime, func() {
		m.DeleteAllocation(a.fiveTuple)
	})

	m.lock.Lock()
	m.allocations[fiveTuple.Fingerprint()] = a
	m.lock.Unlock()

	go a.packetHandler(m)
	return a, nil
}

// DeleteAllocation removes an allocation
func (m *Manager) DeleteAllocation(fiveTuple *FiveTuple) {
	fingerprint := fiveTuple.Fingerprint()

	m.lock.Lock()
	allocation := m.allocations[fingerprint]
	delete(m.allocations, fingerprint)
	m.lock.Unlock()

	if allocation == nil {
		return
	}

	if err := allocation.Close(); err != nil {
		m.log.Errorf("Failed to close allocation: %v", err)
	}
}
