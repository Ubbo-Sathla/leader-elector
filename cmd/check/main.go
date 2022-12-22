package main

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"net"
	"os"
)

func main() {
	vip, _ := netlink.ParseAddr("192.168.1.1/32")

	links, err := netlink.LinkList()
	if err != nil {

	}
	for _, link := range links {
		check := false
		fmt.Printf("%#v\n", link.Attrs().Name)
		address, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {

		}
		fmt.Println(address)

		for _, addr := range address {
			ipStr := os.Getenv("IP")
			ip := net.ParseIP(ipStr)
			if addr.IP.Equal(ip) {
				check = true
			}
		}

		if check {
			fmt.Println("the same")
			err = netlink.AddrAdd(link, vip)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			err = netlink.AddrDel(link, vip)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}
