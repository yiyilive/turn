package turn

import (
	"net"
	"time"

	"github.com/pion/logging"
)

// PacketConnConfig is a single net.PacketConn to listen/write on. This will be used for UDP listeners
type PacketConnConfig struct {
	PacketConn net.PacketConn

	// When an allocation is generated the RelayAddressGenerator
	// creates the net.PacketConn and returns the IP/Port it is available at
	RelayAddressGenerator RelayAddressGenerator
}

func (c *PacketConnConfig) validate() error {
	if c.PacketConn == nil {
		return errConnUnset
	}
	if c.RelayAddressGenerator == nil {
		return errRelayAddressGeneratorUnset
	}

	return nil
}

// ConnConfig is a single net.Conn to listen/write on. This will be used for TCP, TLS and DTLS listeners
type ConnConfig struct {
	Conn net.Conn

	// When an allocation is generated the RelayAddressGenerator
	// creates the net.PacketConn and returns the IP/Port it is available at
	RelayAddressGenerator RelayAddressGenerator
}

func (c *ConnConfig) validate() error {
	if c.Conn == nil {
		return errConnUnset
	}

	if c.RelayAddressGenerator == nil {
		return errRelayAddressGeneratorUnset
	}

	return nil
}

// AuthHandler is a callback used to handle incoming auth requests, allowing users to customize Pion TURN with custom behavior
type AuthHandler func(username string, srcAddr net.Addr) (password string, ok bool)

// ServerConfig configures the Pion TURN Server
type ServerConfig struct {
	// PacketConnConfigs and ConnConfigs are a list of all the turn listeners
	// Each listener can have custom behavior around the creation of Relays
	PacketConnConfigs []PacketConnConfig
	ConnConfigs       []ConnConfig

	// LoggerFactory must be set for logging from this server.
	LoggerFactory logging.LoggerFactory

	// Realm sets the realm for this server
	Realm string

	// AuthHandler is a callback used to handle incoming auth requests, allowing users to customize Pion TURN with custom behavior
	AuthHandler AuthHandler

	// ChannelBindTimeout sets the lifetime of channel binding. Defaults to 10 minutes.
	ChannelBindTimeout time.Duration
}

func (s *ServerConfig) validate() error {
	if len(s.PacketConnConfigs) == 0 && len(s.ConnConfigs) == 0 {
		return errNoAvailableConns
	}

	for _, s := range s.PacketConnConfigs {
		if err := s.validate(); err != nil {
			return err
		}
	}

	for _, s := range s.ConnConfigs {
		if err := s.validate(); err != nil {
			return err
		}
	}

	return nil
}
