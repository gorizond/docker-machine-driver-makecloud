package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/gorizond/docker-machine-driver-makecloud/makecloud"
)

func main() {
	plugin.RegisterDriver(makecloud.NewDriver("", ""))
}
