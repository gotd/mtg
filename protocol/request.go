package protocol

import (
	"context"

	"go.uber.org/zap"

	"github.com/gotd/mtg/conntypes"
)

type TelegramRequest struct {
	Logger         *zap.SugaredLogger
	ClientConn     conntypes.StreamReadWriteCloser
	ConnID         conntypes.ConnID
	Ctx            context.Context
	Cancel         context.CancelFunc
	ClientProtocol ClientProtocol
}
