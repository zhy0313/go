package main

import (
	"github.com/platinasystems/go/elib/parse"
	"github.com/platinasystems/go/vnet"
	"github.com/platinasystems/go/vnet/devices/bus/pci"
	//"github.com/platinasystems/go/vnet/devices/ethernet/ixge"
	"github.com/platinasystems/go/vnet/devices/ethernet/switch/bcm"
	"github.com/platinasystems/go/vnet/ethernet"
	ipcli "github.com/platinasystems/go/vnet/ip/cli"
	"github.com/platinasystems/go/vnet/ip4"
	"github.com/platinasystems/go/vnet/ip6"
	"github.com/platinasystems/go/vnet/pg"
	"github.com/platinasystems/go/vnet/unix"

	"fmt"
	"os"
)

type platform struct {
	vnet.Package
	*bcm.Platform
}

func (p *platform) Init() (err error) {
	v := p.Vnet
	p.Platform = bcm.GetPlatform(v)
	if err = p.boardInit(); err != nil {
		return
	}
	for _, s := range p.Switches {
		if err = p.boardPortInit(s); err != nil {
			return
		}
	}
	return
}

func main() {
	var in parse.Input
	in.Add(os.Args[1:]...)

	v := &vnet.Vnet{}

	// Select packages we want to run with.
	bcm.Init(v)
	ethernet.Init(v)
	ip4.Init(v)
	ip6.Init(v)
	//ixge.Init(v)
	pci.Init(v)
	pg.Init(v)
	ipcli.Init(v)
	unix.Init(v)

	p := &platform{}
	v.AddPackage("platform", p)
	p.DependsOn("pci-discovery") // after pci discovery

	err := v.Run(&in)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
