// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/labstack/echo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/warp-poke/icmp-go-agent/core"
)

var cfgFile string
var verbose bool

var (
	schedulerEventsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "icmp_agent",
		Subsystem: "scheduler",
		Name:      "events_total",
		Help:      "Number of events process by the scheduler",
	})

	schedulerDomainEventsCounterVec = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "icmp_agent",
		Subsystem: "scheduler",
		Name:      "domain_events",
		Help:      "Number of events process by the scheduler",
	}, []string{"domain"})

	schedulerEventsErrorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "icmp_agent",
		Subsystem: "scheduler",
		Name:      "events_error",
		Help:      "Number of events process by the scheduler",
	})
)

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file to use")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	prometheus.MustRegister(schedulerEventsCounter)
	prometheus.MustRegister(schedulerEventsErrorCounter)
	prometheus.MustRegister(schedulerDomainEventsCounterVec)

	viper.BindPFlags(RootCmd.Flags())
}

func initConfig() {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	// Bind environment variables
	viper.SetEnvPrefix("poke_icmp_agent")
	viper.AutomaticEnv()

	viper.SetDefault("timeout", "1s")
	viper.SetDefault("kafka.sasl.enable", false)
	viper.SetDefault("kafka.tls.enable", false)

	viper.SetDefault("metrics.host", "127.0.0.1")
	viper.SetDefault("metrics.port", "9111")

	// Set config search path
	viper.AddConfigPath("/etc/poke-icmp-agent/")
	viper.AddConfigPath("$HOME/.poke-icmp-agent")
	viper.AddConfigPath(".")

	// Load config
	viper.SetConfigName("config")
	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debug("No config file found")
		} else {
			log.Panicf("Fatal error in config file: %v \n", err)
		}
	}

	// Load user defined config
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		err := viper.ReadInConfig()
		if err != nil {
			log.Panicf("Fatal error in config file: %v \n", err)
		}
	}
}

// RootCmd launch the aggregator agent.
var RootCmd = &cobra.Command{
	Use:   "poke-icmp-agent",
	Short: "poke-icmp-agent collect icmp domains stats",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("poke-icmp-agent starting")

		schedulerEvents := core.NewConsumer()

		pingTimeout := viper.GetDuration("timeout")

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)

		metricsServer := echo.New()
		metricsServer.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

		go func() {
			host := viper.GetString("metrics.host")
			port := viper.GetInt("metrics.port")
			if err := metricsServer.Start(fmt.Sprintf("%s:%d", host, port)); err != nil && err != http.ErrServerClosed {
				log.WithError(err).Error("could not start the metrics server")
			}
		}()

		for {
			select {
			case se := <-schedulerEvents:
				schedulerEventsCounter.Inc()
				schedulerDomainEventsCounterVec.WithLabelValues(se.Domain).Inc()
				log.WithFields(log.Fields{
					"event": se,
				}).Info("process new scheduler event")
				if err := core.Process(se, pingTimeout); err != nil {
					schedulerEventsErrorCounter.Inc()
					log.WithError(err).Error("Failed to process scheduler event")
					continue
				}

			case sig := <-quit:
				log.WithField("signal", sig.String()).Println("received signal. exiting")
				if err := metricsServer.Close(); err != nil {
					log.WithError(err).Error("could not stop metrics server")
				}

				os.Exit(1)
			}
		}
	},
}
