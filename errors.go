package turn

import "errors"

var (
	errRelayAddressInvalid        = errors.New("turn: RelayAddress must be valid IP to use RelayAddressGeneratorStatic")
	errNoAvailableConns           = errors.New("turn: PacketConnConfigs and ConnConfigs are empty, unable to proceed")
	errConnUnset                  = errors.New("turn: PacketConnConfig and ConnConfig must have a non-nil Conn")
	errListeningAddressInvalid    = errors.New("turn: RelayAddressGenerator has invalid ListeningAddress")
	errRelayAddressGeneratorUnset = errors.New("turn: RelayAddressGenerator in RelayConfig is unset")
)
