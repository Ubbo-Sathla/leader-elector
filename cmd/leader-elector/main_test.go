package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"syscall"
	"testing"
)

func TestName(t *testing.T) {
	links, err := netlink.LinkList()
	if err != nil {

	}
	for _, link := range links {
		fmt.Println(link.Attrs().Name)
		address, err := netlink.AddrList(link, syscall.IPPROTO_IPV4)
		if err != nil {

		}
		for _, addr := range address {
			fmt.Println(addr)
		}
	}
}
