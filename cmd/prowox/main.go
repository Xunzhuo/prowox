package main

import (
	"time"

	"github.com/xunzhuo/prowox/cmd/prowox/config"
	"github.com/xunzhuo/prowox/pkg/commands"
	"github.com/xunzhuo/prowox/pkg/core"
	"k8s.io/klog"
)

var maxRetries = 60

func main() {
	klog.Info("Starting Prowox ...")

	if err := config.InitConfig(); err != nil {
		klog.Error(err)
		return
	}

	if config.Get().TYPE == "schedule" {
		for i := 0; i < maxRetries; i++ {
			if err := commands.MergeAcceptedPRs(); err != nil {
				klog.Error(err)
			}
			time.Sleep(15 * time.Second)
			klog.Info("Prowox schedule merge in every 15s retry time: ", i+1)
		}
	}

	if config.Get().TYPE == "created" {
		commands.Greeting()
		commands.Help()
		if err := core.RunCommands(); err != nil {
			klog.Error(err)
		}
	}

	if config.Get().TYPE == "comment" {
		if err := core.RunCommands(); err != nil {
			klog.Error(err)
		}
	}
}
