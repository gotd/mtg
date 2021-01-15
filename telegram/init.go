package telegram

import (
	"github.com/9seconds/mtg/conntypes"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

const telegramDialTimeout = 10 * time.Second

var (
	Direct Telegram
	Middle Telegram

	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		logger := zap.S().Named("telegram")

		Direct = CreateDirect(directV4DefaultIdx, directV6DefaultIdx, directV4Addresses, directV6Addresses)

		tg := &middleTelegram{
			baseTelegram: baseTelegram{
				dialer: net.Dialer{Timeout: telegramDialTimeout},
				logger: logger.Named("middle"),
			},
		}
		if err := tg.update(); err != nil {
			panic(err)
		}
		go tg.backgroundUpdate()

		Middle = tg
	})
}

func CreateDirect(defaultDCV4, defaultDCV6 conntypes.DC, v4, v6 map[conntypes.DC][]string) Telegram {
	logger := zap.S().Named("telegram")

	return &directTelegram{
		baseTelegram: baseTelegram{
			dialer:      net.Dialer{Timeout: telegramDialTimeout},
			logger:      logger.Named("direct"),
			v4DefaultDC: defaultDCV4,
			v6DefaultDC: defaultDCV6,
			v4Addresses: v4,
			v6Addresses: v6,
		},
	}
}
