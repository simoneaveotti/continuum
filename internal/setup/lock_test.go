package setup

import "testing"

func TestAcquireStorageLockRejectsConcurrentHolder(t *testing.T) {
	withTempContinuum(t)
	release, err := AcquireStorageLock()
	if err != nil {
		t.Fatalf("AcquireStorageLock() error: %v", err)
	}
	defer release()

	if _, err := AcquireStorageLock(); err == nil {
		t.Fatal("second AcquireStorageLock() unexpectedly succeeded")
	}
}
