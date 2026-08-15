package app

import (
	"sync"
	"testing"
)

// TestVaultStateConcurrency is a data-race regression test for the App state
// mutex (fix 1.2 in Plans/FIXES_TODO.md). Reader goroutines hammer the
// public binding methods (which snapshot vault state) while the test
// goroutine repeatedly swaps the vault in and out under the write lock.
// Run with -race: before the mutex existed, this reliably reported races.
func TestVaultStateConcurrency(t *testing.T) {
	a := newTestApp(t)
	first := a.db
	firstPath := a.vaultPath

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				// Binding methods that snapshot vault state
				_ = a.GetCurrentVault()
				_ = a.GetLastVaultPath()
				_ = a.GetRecentVaults()
				if _, err := a.GetVaultConfig(); err != nil {
					// "no vault open" is expected during swaps
				}
				if _, err := a.GetExcludedFolders(); err != nil {
					// "no vault open" is expected during swaps
				}
				if _, err := a.GetTags(""); err != nil {
					// "no vault open" is expected during swaps
				}
				// Direct snapshot access
				_ = a.vault()
			}
		}()
	}

	// Swap vault state back and forth between two open databases using the
	// same production writer OpenVault uses (setVault): atomic under the
	// write lock. Both DBs are closed by their respective test-app cleanups.
	b := newTestApp(t)
	for i := 0; i < 50; i++ {
		if i%2 == 0 {
			a.setVault(b.db, b.vaultPath, nil)
		} else {
			a.setVault(first, firstPath, nil)
		}
	}

	close(stop)
	wg.Wait()
}
