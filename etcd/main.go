package main

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var EtcdConfig = clientv3.Config{
	Endpoints:   []string{"ec2-3-66-86-193.eu-central-1.compute.amazonaws.com:2379"},
	DialTimeout: 5 * time.Second,
	Username:    "x",
	Password:    "x",
}

var requestTimeout = 2 * time.Second

func (e *etcdLock) Acquire(lockName string, duration time.Duration) bool {

	// create a new session
	s1, err := concurrency.NewSession(e.client)
	if err != nil {
		fmt.Println(err)
	}

	// when session is closed lock on mutex will be released as well
	defer s1.Close()
	m1 := concurrency.NewMutex(s1, lockName)

	// Acquire lock for s1
	if err := m1.Lock(context.TODO()); err != nil {
		fmt.Println("Lock: Couldn't adquire lock")
		fmt.Println(err)
		return false
	}

	fmt.Println("Lock: Got lock for s1")
	time.Sleep(duration * time.Second)
	fmt.Println(*m1)

	return true
}

func (e *etcdLock) TryAcquire(lockName string, duration time.Duration) error {

	// create a new session
	s1, err := concurrency.NewSession(e.client)
	if err != nil {
		fmt.Println(err)
	}
	defer s1.Close()
	m1 := concurrency.NewMutex(s1, lockName)

	// Try acquire lock for s1
	err = m1.TryLock(context.TODO())

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
	fmt.Println(m1)
	time.Sleep(duration * time.Second)
	return err
}

func (e *etcdLock) Release(lockName string) error {

	// create a new session
	s1, err := concurrency.NewSession(e.client)
	if err != nil {
		fmt.Println(err)
	}
	defer s1.Close()

	m1 := concurrency.NewMutex(s1, lockName)

	// Try unlock for s1
	err = m1.Unlock((context.TODO()))
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("Release: released lock for s1")
	return err

}

func (e *etcdLock) Put(key string, value string) (clientv3.PutResponse, error) {

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	out, err := e.client.Put(ctx, key, value)

	cancel()

	if err != nil {
		switch err {
		case context.Canceled:
			fmt.Printf("ctx is canceled by another routine: %v\n", err)
		case context.DeadlineExceeded:
			fmt.Printf("ctx is attached with a deadline is exceeded: %v\n", err)
		case rpctypes.ErrEmptyKey:
			fmt.Printf("client-side error: %v\n", err)
		default:
			fmt.Printf("bad cluster endpoints, which are not etcd servers: %v\n", err)
		}
	}

	return *out, err
}

type etcdLock struct {
	client *clientv3.Client
}

func main() {

	// create client
	cli, err := clientv3.New(EtcdConfig)
	if err != nil {
		fmt.Println(err)
	}
	defer cli.Close()

	// create lock object
	lock := etcdLock{
		cli,
	}

	//lock.TryAcquire("master", 10)
	lock.Acquire("master", 1)

	//time.Sleep(100 * time.Second)

	// when session is closed lock is released
	lock.Release("master")

	// out, _ := Put("tesKey", "TRY123")
	// fmt.Println(out)
}
