package core

import (
	"time"

	warp "github.com/PierreZ/Warp10Exporter"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/warp-poke/icmp-go-agent/models"
)

// Process perform SSL test and return metrics
func Process(e *models.SchedulerEvent, timeout time.Duration) error {

	stats, err := performPing(e.Domain, timeout)

	if err != nil {
		return errors.Wrap(err, "cannot ping")
	}

	now := time.Now()
	batch := warp.NewBatch()

	packetsSentGTS := warp.NewGTS("icmp.packets.sent").WithLabels(warp.Labels{
		"domain": e.Domain,
		"host":   viper.GetString("host"),
		"zone":   viper.GetString("zone"),
		"ip":     stats.IPAddr.String(),
	}).AddDatapoint(now, stats.PacketsSent)
	batch.Register(packetsSentGTS)

	packetsRcvsGTS := warp.NewGTS("icmp.packets.recv").WithLabels(warp.Labels{
		"domain": e.Domain,
		"host":   viper.GetString("host"),
		"zone":   viper.GetString("zone"),
		"ip":     stats.IPAddr.String(),
	}).AddDatapoint(now, stats.PacketsRecv)
	batch.Register(packetsRcvsGTS)

	roundTripGTS := warp.NewGTS("icmp.packets.round.trip.time.ns").WithLabels(warp.Labels{
		"domain": e.Domain,
		"host":   viper.GetString("host"),
		"zone":   viper.GetString("zone"),
		"ip":     stats.IPAddr.String(),
	})
	if len(stats.Rtts) == 1 {
		roundTripGTS.AddDatapoint(now, stats.Rtts[0].Nanoseconds())
		batch.Register(roundTripGTS)
	}

	return batch.Push(e.Warp10Endpoint, e.Token)
}
