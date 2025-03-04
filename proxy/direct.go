package proxy

import (
	"io"
	"sync"

	"go.uber.org/zap"

	"github.com/gotd/mtg/conntypes"
	"github.com/gotd/mtg/obfuscated2"
	"github.com/gotd/mtg/protocol"
	"github.com/gotd/mtg/telegram"
)

const directPipeBufferSize = 1024

func directConnection(dialer telegram.Telegram, request *protocol.TelegramRequest) error {
	telegramConnRaw, err := obfuscated2.TelegramProtocolWithDialer(dialer, request)
	if err != nil {
		return err
	}

	telegramConn := telegramConnRaw.(conntypes.StreamReadWriteCloser)

	defer telegramConn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go directPipe(telegramConn, request.ClientConn, wg, request.Logger)

	go directPipe(request.ClientConn, telegramConn, wg, request.Logger)

	wg.Wait()

	return nil
}

func directPipe(dst io.WriteCloser, src io.ReadCloser, wg *sync.WaitGroup, logger *zap.SugaredLogger) {
	defer func() {
		dst.Close()
		src.Close()
		wg.Done()
	}()

	buf := [directPipeBufferSize]byte{}

	if _, err := io.CopyBuffer(dst, src, buf[:]); err != nil {
		logger.Debugw("Cannot pump sockets", "error", err)
	}
}
