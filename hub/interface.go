package hub

import "github.com/gotd/mtg/protocol"

type Interface interface {
	Register(*protocol.TelegramRequest) (*ProxyConn, error)
}
