package cli

import (
	"net"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gotd/mtg/antireplay"
	"github.com/gotd/mtg/config"
	"github.com/gotd/mtg/faketls"
	"github.com/gotd/mtg/hub"
	"github.com/gotd/mtg/ntp"
	"github.com/gotd/mtg/obfuscated2"
	"github.com/gotd/mtg/proxy"
	"github.com/gotd/mtg/stats"
	"github.com/gotd/mtg/telegram"
	"github.com/gotd/mtg/utils"
)

func Proxy() error { // nolint: funlen
	ctx := utils.GetSignalContext()

	atom := zap.NewAtomicLevel()

	switch {
	case config.C.Debug:
		atom.SetLevel(zapcore.DebugLevel)
	case config.C.Verbose:
		atom.SetLevel(zapcore.InfoLevel)
	default:
		atom.SetLevel(zapcore.ErrorLevel)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stderr),
		atom,
	))

	zap.ReplaceGlobals(logger)
	defer logger.Sync() // nolint: errcheck

	if err := config.InitPublicAddress(ctx); err != nil {
		Fatal(err)
	}

	zap.S().Debugw("Configuration", "config", config.Printable())

	if config.C.MiddleProxyMode() {
		zap.S().Infow("Use middle proxy connection to Telegram")

		diff, err := ntp.Fetch()
		if err != nil {
			Fatal("Cannot fetch time data from NTP")
		}

		if diff > time.Second {
			Fatal("Your local time is skewed and drift is bigger than a second. Please sync your time.")
		}

		go ntp.AutoUpdate()
	} else {
		zap.S().Infow("Use direct connection to Telegram")
	}

	PrintJSONStdout(config.GetURLs())

	if err := stats.Init(ctx); err != nil {
		Fatal(err)
	}

	antireplay.Init()
	telegram.Init()
	hub.Init(ctx)

	proxyListener, err := net.Listen("tcp", config.C.Bind.String())
	if err != nil {
		Fatal(err)
	}

	go func() {
		<-ctx.Done()
		proxyListener.Close()
	}()

	app := &proxy.Proxy{
		Logger:              zap.S().Named("proxy"),
		Context:             ctx,
		ClientProtocolMaker: obfuscated2.MakeClientProtocol,
	}
	if config.C.SecretMode == config.SecretModeTLS {
		app.ClientProtocolMaker = faketls.MakeClientProtocol
	}

	app.Serve(proxyListener)

	return nil
}
