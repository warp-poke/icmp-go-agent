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
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/warp-poke/icmp-go-agent/core"
)

var cfgFile string
var verbose bool

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file to use")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

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

		for {
			select {
			case se := <-schedulerEvents:
				log.WithFields(log.Fields{
					"event": se,
				}).Info("process new scheduler event")
				if err := core.Process(se, pingTimeout); err != nil {
					log.WithError(err).Error("Failed to process scheduler event")
					continue
				}

			case sig := <-quit:
				log.WithField("signal", sig.String()).Println("received signal. exiting")
				os.Exit(1)
			}
		}
	},
}
