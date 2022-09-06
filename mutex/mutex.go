package mutex

import (
	"context"

	"github.com/rs/zerolog/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type Lock struct {
	Client  *clientv3.Client
	Session *concurrency.Session
	Mutex   *concurrency.Mutex
}

func (e *Lock) Acquire() error {
	// Acquire lock for s1
	err := e.Mutex.Lock(context.TODO())

	if err != nil {
		log.Error().Msgf("%s", err)
		return err
	}

	log.Debug().Msgf("Lock: got lock %s", e.Mutex.Key())
	return err
}

// Close releases the client and session resources
func (e *Lock) Close() error {
	err := e.Session.Close()
	if err != nil {
		log.Error().Msgf("%s", err)
		return err
	}
	err = e.Client.Close()
	if err != nil {
		log.Error().Msgf("%s", err)
	}
	return err
}

func (e *Lock) TryAcquire() error {
	// Try acquire lock for s1
	err := e.Mutex.TryLock(context.TODO())

	if err != nil {
		log.Debug().Msgf("TryLock: Couldn't adquire lock %s", e.Mutex.Key())
		switch err {
		case concurrency.ErrLocked:
			log.Error().Msg("cannot acquire lock, as already locked in another session")
		default:
			log.Error().Msgf("%s", err)
		}
		return err
	}
	log.Debug().Msgf("TryLock: got lock %s", e.Mutex.Key())
	return err
}

func (e *Lock) Release() error {
	log.Debug().Msgf("releasing lock: %s", e.Mutex.Key())
	err := e.Mutex.Unlock((context.TODO()))
	if err != nil {
		log.Error().Msgf("%s", err)
	}
	return err
}
