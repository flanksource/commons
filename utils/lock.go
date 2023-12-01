package utils

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type Unlocker interface {
	Release()
}

type unlocker struct {
	lock *semaphore.Weighted
}

func (u *unlocker) Release() {
	u.lock.Release(1)
}

type NamedLock struct {
	locks sync.Map
}

func (n *NamedLock) TryLock(name string, timeout time.Duration) Unlocker {
	o, _ := n.locks.LoadOrStore(name, semaphore.NewWeighted(1))
	lock := o.(*semaphore.Weighted)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	if err := lock.Acquire(ctx, 1); err != nil {
		return nil
	}
	return &unlocker{lock}
}
