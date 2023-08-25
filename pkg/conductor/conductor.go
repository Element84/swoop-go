package conductor

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/element84/swoop-go/pkg/config"
	"github.com/element84/swoop-go/pkg/db"
	"github.com/element84/swoop-go/pkg/s3"
)

type PgConductor struct {
	InstanceName string
	S3Driver     *s3.S3Driver
	SwoopConfig  *config.SwoopConfig
	DbConfig     *db.ConnectConfig
}

func (c *PgConductor) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(
		&c.InstanceName,
		"conductor-instance",
		"n",
		"",
		"conductor instance name (required; SWOOP_CONDUCTOR_INSTANCE)",
	)
	cobra.MarkFlagRequired(fs, "conductor-instance")
}

func (c *PgConductor) Run(ctx context.Context, cancel context.CancelFunc) error {
	// init handlers
	conf, ok := c.SwoopConfig.Conductors[c.InstanceName]
	if !ok {
		return fmt.Errorf("no conductor config for instance '%s'", c.InstanceName)
	}

	handlerConfs := conf.Handlers
	if len(handlerConfs) == 0 {
		return fmt.Errorf("no handlers specified for conductor instance '%s'", c.InstanceName)
	}

	handlers := []*Handler{}
	for _, conf := range handlerConfs {
		handler, err := NewHandlerFromConfig(ctx, conf)
		if err != nil {
			// TODO: I think this should be an error, not just logged?
			log.Println(err)
			continue
		}

		handlers = append(handlers, handler)
	}

	// start listening
	// TODO: how to keep it listening, maybe with backoff?
	err := db.Listen(ctx, c.DbConfig, handlers)
	if err != nil {
		return err
	}

	// start handlers
	// we start handlers after we start listening
	// to ensure we don't miss any nofications
	var wg sync.WaitGroup
	for _, handler := range handlers {
		handler := handler
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.Start(ctx, c.DbConfig)
		}()
	}

	wg.Wait()

	return nil
}

func (c *PgConductor) SignalHandler(
	signalChan <-chan os.Signal,
	ctx context.Context,
	cancel context.CancelFunc,
) {
	select {
	case sig := <-signalChan:
		switch sig {
		case syscall.SIGINT:
			log.Printf("Got SIGINT, exiting.")
			cancel()
		case syscall.SIGTERM:
			log.Printf("Got SIGTERM, exiting.")
			cancel()
		}
	case <-ctx.Done():
		log.Printf("Done.")
	}
}
