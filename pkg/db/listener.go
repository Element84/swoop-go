package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Notifiable interface {
	GetName() string
	Notify(msg string)
}

func listen(ctx context.Context, conn *pgx.Conn, notifMap map[string]Notifiable) {
	defer conn.Close(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			log.Printf("error while waiting for pg notification: %s", err)
			if conn.IsClosed() {
				// any fatal errors will close the connection per
				// https://github.com/jackc/pgx/blob/8fb309c6317483733c783e9f9a4ac09cb8271849/pgconn/pgconn.go#L515
				return
			}
			continue
		}

		notifiable, ok := notifMap[notification.Channel]
		if !ok {
			log.Printf("notification received for unknown channel '%s'", notification.Channel)
			continue
		}

		notifiable.Notify(notification.Payload)
	}
}

func Listen(ctx context.Context, config *ConnectConfig, notifiables []Notifiable) error {
	listening := false

	if len(notifiables) == 0 {
		return fmt.Errorf("not listening: nothing to listen to")
	}

	conn, err := config.Connect(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if !listening {
			conn.Close(ctx)
		}
	}()

	notifMap := map[string]Notifiable{}
	sqlStmts := []string{}
	for _, notifiable := range notifiables {
		name := notifiable.GetName()
		notifMap[name] = notifiable
		sqlStmts = append(sqlStmts, fmt.Sprintf(`LISTEN "%s";`, name))
	}

	_, err = conn.Exec(ctx, strings.Join(sqlStmts[:], "\n"))
	if err != nil {
		// TODO: abstract this error handling and use elsewhere
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			log.Printf("%+v", err)
		}
		return err
	}

	go listen(ctx, conn, notifMap)
	listening = true

	return nil
}
