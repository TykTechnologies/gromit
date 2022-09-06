package mutex

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3/concurrency"
	"go.etcd.io/etcd/client/v3"
)

type Lock struct {
	Client *clientv3.Client
	Session *concurrency.Session
	Mutex *concurrency.Mutex
}

func (e *Lock) Acquire() error {
	// Acquire lock for s1
	if err := e.Mutex.Lock(context.TODO()); err != nil {
		fmt.Println("Lock: Couldn't adquire lock")
		fmt.Println(err)
		return err
	}

	fmt.Println("Lock: Got lock for s1")
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

func (e *Lock) TryAcquire(lockName string, duration time.Duration) error {
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

	fmt.Println("Got Lock!")
	time.Sleep(duration * time.Second)
	return err
}

func (e *Lock) Release() error {
	err := e.Mutex.Unlock((context.TODO()))
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Release: released lock for s1")
	return err
}
