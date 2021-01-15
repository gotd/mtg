package telegram

import "github.com/gotd/mtg/conntypes"

type Telegram interface {
	Dial(conntypes.DC, conntypes.ConnectionProtocol) (conntypes.StreamReadWriteCloser, error)
	Secret() []byte
}
