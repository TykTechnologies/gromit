package mutex

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/client/v3/concurrency"
)

const TestLockName = "ConcurrentTest"

// TestConcurrentLocking starts a simulated process after taking a lock, and
// then waits a bit to let this process start off comfortably  and then
// starts another process parallelly with the same lock.
// The first process should complete successfully, and the second one should
// return with the already locked error.
func TestConcurrentLocking(t *testing.T) {
	e1 := make(chan error)
	e2 := make(chan error)
	go processWithLockChan(t, e1)
	time.Sleep(1 * time.Second)
	go processWithLockChan(t, e2)
	assert.Equal(t, <-e1, nil)
	assert.EqualError(t, <-e2, concurrency.ErrLocked.Error())
}

// TestSubsequentLockingWithSameLock starts a simulated process, and waits
// for it to finish and then starts another process simulation.
// Both should finish successfully without any errors.
func TestSubsequentLockingWithSameLock(t *testing.T) {
	e1 := processWithLock(t)
	e2 := processWithLock(t)
	assert.Equal(t, e1, nil)
	assert.Equal(t, e2, nil)
}

func processWithLock(t *testing.T) error {
	user := os.Getenv("TEST_ETCD_USER")
	pw := os.Getenv("TEST_ETCD_PASS")
	host := os.Getenv("TEST_ETCD_HOST")
	if host == "" || pw == "" || user == "" {
		t.Error("Unable to get ETCD endpoints and/or credentials.")
		return errors.New("no-env-vars")
	}
	client, err := GetEtcdClient(host, 5, user, pw)
	if err != nil {
		t.Error("Unable to create etcd client: ", err)
		return err
	}
	session, err := GetSession(client)
	if err != nil {
		t.Error("Unable to get etcd session lease: ", err)
		return err
	}
	mtx := GetMutex(session, TestMutexPrefix+TestLockName)
	lock := Lock{
		Client:  client,
		Session: session,
		Mutex:   mtx,
	}
	err = lock.TryAcquire()
	if err != nil {
		t.Log("Environment being created by another process.")
		return err
	}
	t.Log("Simulating some processing.")
	time.Sleep(10 * time.Second)
	lock.Release()
	return nil
}

func processWithLockChan(t *testing.T, status chan<- error) {
	err := processWithLock(t)
	status <- err
}
