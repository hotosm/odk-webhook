package db

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/matryer/is"
)

// NB: these tests assume you have a postgres server listening on localhost:5432
// with username postgres and password postgres. You can trivially set this up
// with Docker with the following:
//
// docker run --rm --name postgres -p 5432:5432 \
// -e POSTGRES_PASSWORD=postgres postgres

func TestNotifier(t *testing.T) {
	// TODO this should be a local db in a compose stack
	dbUri := "postgresql://odk:odk@host.docker.internal:5434/odk?sslmode=disable"

	is := is.New(t)
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}
	pool, err := InitPool(ctx, log, dbUri)
	is.NoErr(err)

	listener := NewListener(pool)
	err = listener.Connect(ctx)
	is.NoErr(err)

	n := NewNotifier(log, listener)
	wg.Add(1)
	go func() {
		n.Run(ctx)
		wg.Done()
	}()
	sub := n.Listen("foo")

	conn, err := pool.Acquire(ctx)
	wg.Add(1)
	go func() {
		<-sub.EstablishedC()
		conn.Exec(ctx, "select pg_notify('foo', '1')")
		conn.Exec(ctx, "select pg_notify('foo', '2')")
		conn.Exec(ctx, "select pg_notify('foo', '3')")
		conn.Exec(ctx, "select pg_notify('foo', '4')")
		conn.Exec(ctx, "select pg_notify('foo', '5')")
		wg.Done()
	}()
	is.NoErr(err)

	wg.Add(1)

	out := make(chan string)
	go func() {
		<-sub.EstablishedC()
		for i := 0; i < 5; i++ {
			msg := <-sub.NotificationC()
			out <- string(msg)
		}
		close(out)
		wg.Done()
	}()

	msgs := []string{}
	for r := range out {
		msgs = append(msgs, r)
	}
	is.Equal(msgs, []string{"1", "2", "3", "4", "5"})

	cancel()
	sub.Unlisten(ctx) // uses background ctx anyway
	listener.Close(ctx)
	wg.Wait()
}
