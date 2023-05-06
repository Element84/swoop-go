package cmdutil

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type CliHandler interface {
	SignalHandler(
		signalChan <-chan os.Signal,
		ctx context.Context,
		cancel context.CancelFunc,
	)
	Run(ctx context.Context, cancel context.CancelFunc) error
}

func Run(c CliHandler) error {
	ctx := context.Background()
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
