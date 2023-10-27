package vip

import (
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

// Network is an interface that enable managing operations for a given IP
type Network interface {
	AddIP() error
	DeleteIP() error
	IsSet() (bool, error)
	Interface() string
	IsMaster() bool
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

// IsMaster - Check to see if VIP is set
func (configurator *network) IsMaster() (result bool) {
	return configurator.link.Attrs().MasterIndex == 0
}

func NewConfig(address string, slave string, master string) (Network, error) {
	result := &network{}
	var err error
	var link netlink.Link

	result.address, err = netlink.ParseAddr(address)
	if err != nil {
		return result, errors.Wrapf(err, "could not parse address '%s'", address)
	}

	link, err = netlink.LinkByName(slave)
	if err != nil {
		return result, errors.Wrapf(err, "could not get link for interface '%s'", slave)
	}

	result.link = link

	result.DeleteIP()

	if !result.IsMaster() {
		link, err = netlink.LinkByName(master)
		if err != nil {
			return result, errors.Wrapf(err, "could not get link for interface '%s'", slave)
		}
		result.link = link

	}
	result.DeleteIP()

	return result, err
}
