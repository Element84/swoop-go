package cmdutil

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	scontext "github.com/element84/swoop-go/pkg/context"
)

type CliHandler interface {
	SignalHandler(
		signalChan <-chan os.Signal,
		ctx context.Context,
		cancel context.CancelFunc,
	)
	Run(ctx context.Context, cancel context.CancelFunc) error
}

func Run(appName string, c CliHandler) error {
	ctx := scontext.NewApplicationContext(appName)
	ctx, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go c.SignalHandler(signalChan, ctx, cancel)

	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	return c.Run(ctx, cancel)
}
