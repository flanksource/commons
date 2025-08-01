package utils

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Unlocker provides a method to release a lock.
// It follows the standard pattern for lock guards in Go.
type Unlocker interface {
	Release()
}

type unlocker struct {
	lock *semaphore.Weighted
}

func (u *unlocker) Release() {
	u.lock.Release(1)
}

// NamedLock provides a mechanism for acquiring locks by name, allowing
// different parts of the code to synchronize on string identifiers.
// This is useful for resource-based locking where you want to ensure
// exclusive access to resources identified by strings (e.g., file paths,
// user IDs, resource names).
//
// The implementation uses semaphores internally, providing timeout support
// and preventing goroutine leaks.
type NamedLock struct {
	locks sync.Map
}

// TryLock attempts to acquire a lock with the given name within the specified timeout.
// If successful, it returns an Unlocker that must be used to release the lock.
// If the lock cannot be acquired within the timeout, it returns nil.
//
// The lock is exclusive - only one goroutine can hold a lock with a given name
// at any time.
//
// Example:
//
//	lock := &NamedLock{}
//	
//	// Try to acquire lock with 5-second timeout
//	if unlocker := lock.TryLock("user-123", 5*time.Second); unlocker != nil {
//		defer unlocker.Release()
//		// Critical section - exclusive access to user-123
//		updateUser("123")
//	} else {
//		// Could not acquire lock within timeout
//		return errors.New("resource is locked")
//	}
//
// Multiple locks example:
//
//	// Different names don't block each other
//	go func() {
//		if u := lock.TryLock("resource-A", time.Second); u != nil {
//			defer u.Release()
//			// Work with resource A
//		}
//	}()
//	
//	go func() {
//		if u := lock.TryLock("resource-B", time.Second); u != nil {
//			defer u.Release()
//			// Work with resource B
//		}
//	}()
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
