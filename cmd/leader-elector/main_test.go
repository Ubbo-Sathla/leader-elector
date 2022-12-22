package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"testing"
)

func TestName(t *testing.T) {
	links, err := netlink.LinkList()
	if err != nil {

	}
	for _, link := range links {
		t.Log(link.Attrs().Name)
		address, err := netlink.AddrList(link, 4)
		if err != nil {

		}
		for _, addr := range address {
			fmt.Println(addr)
			t.Log(addr)
		}
	}
}
