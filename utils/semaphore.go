package utils

import "sync"

// TODO: Use KVS (eg. Redis) or Adopt a method that does not require using semaphores

var (
	SemaphoreControlPlane sync.Mutex
)
