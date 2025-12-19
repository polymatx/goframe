package signal

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitExitSignal blocks until an interrupt or terminate signal is received
// Returns the signal that was received
func WaitExitSignal() os.Signal {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	return sig
}

// WaitExitSignalWithCallback waits for exit signal and calls cleanup function
func WaitExitSignalWithCallback(cleanup func()) os.Signal {
	sig := WaitExitSignal()
	if cleanup != nil {
		cleanup()
	}
	return sig
}
