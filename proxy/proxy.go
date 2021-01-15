package proxy

import (
	"context"
	"net"

	"go.uber.org/zap"

	"github.com/gotd/mtg/config"
	"github.com/gotd/mtg/conntypes"
	"github.com/gotd/mtg/protocol"
	"github.com/gotd/mtg/stats"
	"github.com/gotd/mtg/telegram"
	"github.com/gotd/mtg/utils"
	"github.com/gotd/mtg/wrappers/stream"
)

type Proxy struct {
	Logger              *zap.SugaredLogger
	Context             context.Context
	ClientProtocolMaker protocol.ClientProtocolMaker
}

func (p *Proxy) Serve(listener net.Listener) {
	doneChan := p.Context.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-doneChan:
				return
			default:
				p.Logger.Fatalw("Cannot allocate incoming connection", "error", err)
			}
		}

		go p.accept(conn)
	}
}

func (p *Proxy) accept(conn net.Conn) {
	defer func() {
		conn.Close()

		if err := recover(); err != nil {
			stats.Stats.Crash()
			p.Logger.Errorw("Crash of accept handler", "error", err)
		}
	}()

	connID := conntypes.NewConnID()
	logger := p.Logger.With("connection_id", connID)

	if err := utils.InitTCP(conn, config.C.ClientReadBuffer(), config.C.ClientWriteBuffer()); err != nil {
		logger.Errorw("Cannot initialize client TCP connection", "error", err)

		return
	}

	ctx, cancel := context.WithCancel(p.Context)
	defer cancel()

	clientConn := stream.NewClientConn(conn, connID)
	clientConn = stream.NewCtx(ctx, cancel, clientConn)
	clientConn = stream.NewTimeout(clientConn)

	defer clientConn.Close()

	clientProtocol := p.ClientProtocolMaker()

	clientConn, err := clientProtocol.Handshake(clientConn)
	if err != nil {
		stats.Stats.AuthenticationFailed()
		logger.Warnw("Cannot perform client handshake", "error", err)

		return
	}

	stats.Stats.ClientConnected(clientProtocol.ConnectionType(), clientConn.RemoteAddr())
	defer stats.Stats.ClientDisconnected(clientProtocol.ConnectionType(), clientConn.RemoteAddr())
	logger.Infow("Client connected", "addr", conn.RemoteAddr())

	req := &protocol.TelegramRequest{
		Logger:         logger,
		ClientConn:     clientConn,
		ConnID:         connID,
		Ctx:            ctx,
		Cancel:         cancel,
		ClientProtocol: clientProtocol,
	}

	if config.C.MiddleProxyMode() {
		middleConnection(req)
	} else {
		dialer := telegram.Direct
		if config.C.TestDC {
			dialer = telegram.CreateDirect(2, 2, map[conntypes.DC][]string{
				2: {"149.154.167.40:443"},
			}, map[conntypes.DC][]string{})
		}
		err = directConnection(dialer, req)
	}

	logger.Infow("Client disconnected", "error", err, "addr", conn.RemoteAddr())
}
