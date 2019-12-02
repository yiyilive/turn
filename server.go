package turn

import (
	"net"
	"time"

	"github.com/pion/logging"
	"github.com/pion/turn/internal/allocation"
	"github.com/pion/turn/internal/proto"
	"github.com/pion/turn/internal/server"
)

const (
	inboundMTU = 1500
)

// Server is an instance of the Pion TURN Server
type Server struct {
	log                logging.LeveledLogger
	authHandler        AuthHandler
	realm              string
	channelBindTimeout time.Duration
}

// NewServer creates the Pion TURN server
func NewServer(config ServerConfig) (*Server, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	s := &Server{
		log:                config.LoggerFactory.NewLogger("turn"),
		authHandler:        config.AuthHandler,
		realm:              config.Realm,
		channelBindTimeout: config.ChannelBindTimeout,
	}

	if s.channelBindTimeout == 0 {
		s.channelBindTimeout = proto.DefaultLifetime
	}

	for _, p := range config.PacketConnConfigs {
		go s.connReadLoop(p.PacketConn, p.RelayAddressGenerator)
	}

	return s, nil
}

// Close stops the TURN Server. It cleans up any associated state and closes all connections it is managing
func (s *Server) Close() error {
	// TODO Close all sockets, block until all the reads are done
	return nil
}

func (s *Server) connReadLoop(p net.PacketConn, r RelayAddressGenerator) {
	buf := make([]byte, inboundMTU)
	allocationManager, err := allocation.NewManager(allocation.ManagerConfig{
		RelayAddressGenerator: r.Allocate,
		LeveledLogger:         s.log,
	})
	if err != nil {
		s.log.Errorf("exit read loop on error: %s", err.Error())
		return
	}

	for {
		n, addr, err := p.ReadFrom(buf)
		if err != nil {
			s.log.Debugf("exit read loop on error: %s", err.Error())
			return
		}

		if err := server.HandleRequest(server.Request{
			Conn:               p,
			SrcAddr:            addr,
			Buff:               buf[:n],
			Log:                s.log,
			AuthHandler:        s.authHandler,
			Realm:              s.realm,
			AllocationManager:  allocationManager,
			ChannelBindTimeout: s.channelBindTimeout,
		}); err != nil {
			s.log.Errorf("error when handling datagram: %v", err)
		}
	}
}
