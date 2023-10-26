package vip

import (
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Network is an interface that enable managing operations for a given IP
type Network interface {
	AddIP() error
	DeleteIP() error
	IsSet() (bool, error)
	Interface() string
}

// network - This allows network configuration
type network struct {
	address *netlink.Addr
	link    netlink.Link
}

func netlinkParse(addr string) (*netlink.Addr, error) {
	mask, err := GetFullMask(addr)
	if err != nil {
		return nil, err
	}
	return netlink.ParseAddr(addr + mask)
}

// Interface - return the Interface name
func (configurator *network) Interface() string {
	return configurator.link.Attrs().Name
}

// IsSet - Check to see if VIP is set
func (configurator *network) IsSet() (result bool, err error) {
	var addresses []netlink.Addr

	if configurator.address == nil {
		return false, nil
	}

	addresses, err = netlink.AddrList(configurator.link, 0)
	if err != nil {
		err = errors.Wrap(err, "could not list addresses")

		return
	}

	for _, address := range addresses {
		if address.Equal(*configurator.address) {
			return true, nil
		}
	}

	return false, nil
}

// DeleteIP - Remove an IP address from the interface
func (configurator *network) DeleteIP() error {
	result, err := configurator.IsSet()
	if err != nil {
		return errors.Wrap(err, "ip check in DeleteIP failed")
	}

	// Nothing to delete
	if !result {
		return nil
	}

	if err = netlink.AddrDel(configurator.link, configurator.address); err != nil {
		return errors.Wrap(err, "could not delete ip")
	}

	return nil
}

// AddIP - Add an IP address to the interface
func (configurator *network) AddIP() error {
	if err := netlink.AddrReplace(configurator.link, configurator.address); err != nil {
		return errors.Wrap(err, "could not add ip")
	}

	return nil
}

func NewConfig(address string, iface string, subnet string, isDDNS bool, tableID int) (Network, error) {
	result := &network{}

	link, err := netlink.LinkByName(iface)
	if err != nil {
		return result, errors.Wrapf(err, "could not get link for interface '%s'", iface)
	}

	result.link = link

	if IsIP(address) {
		// Check if the subnet needs overriding
		if subnet != "" {
			result.address, err = netlink.ParseAddr(address + subnet)
			if err != nil {
				return result, errors.Wrapf(err, "could not parse address '%s'", address)
			}
		} else {
			result.address, err = netlinkParse(address)
			if err != nil {
				return result, errors.Wrapf(err, "could not parse address '%s'", address)
			}
		}
		// Ensure we don't have a global address on loopback
		if iface == "lo" {
			result.address.Scope = unix.RT_SCOPE_HOST
		}
		return result, nil
	}

	return result, err
}
