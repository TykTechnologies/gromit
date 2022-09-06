package mutex

import (
	"context"
	"fmt"

	"go.etcd.io/etcd/client/v3/concurrency"
	"go.etcd.io/etcd/client/v3"
	"github.com/rs/zerolog/log"
)

type Lock struct {
	Client *clientv3.Client
	Session *concurrency.Session
	Mutex *concurrency.Mutex
}

func (e *Lock) Acquire() error {
	// Acquire lock for s1
	if err := e.Mutex.Lock(context.TODO()); err != nil {
		return err
	}

	log.Debug().Msgf("Lock: got lock %s", e.Mutex.Key())
	return nil
}

// Close releases the client and session resources
func (e *Lock) Close() error {
	err := e.Session.Close()
	if err != nil {
		return err
	}
	err = e.Client.Close()
	return err
}

func (e *Lock) TryAcquire() error {
	// Try acquire lock for s1
	err := e.Mutex.TryLock(context.TODO())

	if err != nil {
		fmt.Println("TryLock: Couldn't adquire lock")
		switch err {
		case concurrency.ErrLocked:
			fmt.Println("cannot acquire lock, as already locked in another session")
		default:
			fmt.Println(err)
		}
		return err
	}
	log.Debug().Msgf("TryLock: got lock %s", e.Mutex.Key())
	return err
}

func (e *Lock) Release() error {
	log.Debug().Msgf("releasing lock: %s", e.Mutex.Key())
	err := e.Mutex.Unlock((context.TODO()))

	return err
}
