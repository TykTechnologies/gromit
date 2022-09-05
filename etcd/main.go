package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var EtcdConfig = clientv3.Config{
	Endpoints:   []string{"ec2-3-66-86-193.eu-central-1.compute.amazonaws.com:2379"},
	DialTimeout: 5 * time.Second,
	Username:    "xxxxxxxxx",
	Password:    "xxxxxxxxx",
}

var requestTimeout = 4 * time.Second

func (e *etcdLock) Acquire(lockName string) *concurrency.Mutex {

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
		switch err {
		case concurrency.ErrLocked:
			fmt.Println("cannot acquire lock, as already locked in another session")
			return m1
		default:
			fmt.Println(err)
			return m1
		}
	}
	fmt.Println(m1)
	return m1

}

func (e *etcdLock) Release(lockName string) {

}

type etcdLock struct {
	client *clientv3.Client
}

func main() {

	out, _ := Put("tesKey", "TRY123")
	fmt.Println(out)

	cli, err := clientv3.New(EtcdConfig)
	if err != nil {
		fmt.Println(err)
	}
	defer cli.Close()

	lock := etcdLock{
		cli,
	}

	lock.Acquire("master")

	time.Sleep(30 * time.Second)

	// mtx := AdquireLock("myLock")
	// fmt.Println(mtx)

	// cli, err := clientv3.New(EtcdConfig)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// defer cli.Close()

	// if err = mtx.Unlock(context.TODO()); err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println("Context unlocked??")
	// fmt.Println(mtx)
	// fmt.Println("The End")

}

// func AdquireLock(client clientv3, lockName string) concurrency.Mutex {

// 	cli, err := clientv3.New(EtcdConfig)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer cli.Close()

// 	// create a new session
// 	s1, err := concurrency.NewSession(cli)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer s1.Close()
// 	m1 := concurrency.NewMutex(s1, lockName)

// 	// Try acquire lock for s1
// 	err = m1.TryLock(context.TODO())
// 	if err != nil {
// 		switch err {
// 		case concurrency.ErrLocked:
// 			fmt.Println("cannot acquire lock, as already locked in another session")
// 			return *m1
// 		default:
// 			fmt.Println(err)
// 			return *m1
// 		}
// 	}
// 	fmt.Println(*m1)
// 	return *m1
// }

// func ReleaseLock(lockName string) concurrency.Mutex {

// 	cli, err := clientv3.New(EtcdConfig)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer cli.Close()

// 	// create a new session
// 	s1, err := concurrency.NewSession(cli)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer s1.Close()
// 	m1 := concurrency.NewMutex(s1, lockName)

// 	// Try unlock for s1
// 	err = m1.Unlock((context.TODO()))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Println("released lock for s1")
// 	fmt.Println(*m1)
// 	return *m1
// }

func Put(key string, value string) (clientv3.PutResponse, error) {
	cli, err := clientv3.New(EtcdConfig)

	if err != nil {
		log.Fatal(err)
	}

	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	out, err := cli.Put(ctx, key, value)

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
