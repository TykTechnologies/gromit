package mutex

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/rs/zerolog/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// ProdMutexPrefix is the prefix for all the created mutex objects.
const ProdMutexPrefix = "/gromit/prod/"

// TestMutexPrefix should only be used from within unit and integration
// tests.
const TestMutexPrefix = "/gromit/test/"

type Lock struct {
	Client  *clientv3.Client
	Session *concurrency.Session
	Mutex   *concurrency.Mutex
}

// GetEtcdClient creates an etcd client.
func GetEtcdClient(host string, timeout time.Duration, user string, pass string) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   []string{host},
		DialTimeout: timeout * time.Second,
		Username:    user,
		Password:    pass,
	})
}

func GetEtcdTLSClient(host string, tls *tls.Config, timeout time.Duration) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   []string{host},
		DialTimeout: timeout * time.Second,
		TLS:         tls,
	})
}

func GetSession(client *clientv3.Client) (*concurrency.Session, error) {
	return concurrency.NewSession(client)
}

func GetMutex(session *concurrency.Session, prefix string) *concurrency.Mutex {
	return concurrency.NewMutex(session, prefix)
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

// Try to adquire a lock (non-blocking), if process is locked will log error and exit without locking
func (e *Lock) TryAcquire() error {
	// Try acquire lock for s1
	err := e.Mutex.TryLock(context.TODO())

	if err != nil {
		log.Debug().Msgf("TryLock: Couldn't adquire lock: %s", e.Mutex.Key())
		switch err {
		case concurrency.ErrLocked:
			log.Error().Msg("cannot acquire lock, as already locked in another session")
		default:
			log.Error().Msgf("%s", err)
		}
		return err
	}
	log.Debug().Msgf("TryLock: got lock: %s", e.Mutex.Key())
	return err
}

// Release an existent lock
func (e *Lock) Release() error {
	log.Debug().Msgf("releasing lock: %s", e.Mutex.Key())
	err := e.Mutex.Unlock((context.TODO()))
	if err != nil {
		log.Error().Msgf("%s", err)
	}
	return err
}
