package core

import (
	"time"

	"github.com/pkg/errors"
	ping "github.com/sparrc/go-ping"
)

func performPing(destination string, timeout time.Duration) (*ping.Statistics, error) {

	pinger, err := ping.NewPinger(destination)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create pinger")
	}

	pinger.Count = 1
	pinger.Timeout = timeout
	pinger.Run() // blocks until finished

	return pinger.Statistics(), nil
}
