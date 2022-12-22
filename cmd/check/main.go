package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
)

func main() {
	links, err := netlink.LinkList()
	if err != nil {

	}
	for _, link := range links {
		fmt.Printf("%#v\n", link.Attrs().Name)
		address, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {

		}
		for _, addr := range address {
			fmt.Println(addr)
		}
	}
}
