package vip

import (
	"fmt"
	"net"
)

// IsIP returns if address is an IP or not
func IsIP(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil
}

// IsIPv4 returns true only if address is a valid IPv4 address
func IsIPv4(address string) bool {
	ip := net.ParseIP(address)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}

// IsIPv6 returns true only if address is a valid IPv6 address
func IsIPv6(address string) bool {
	ip := net.ParseIP(address)
	if ip == nil {
		return false
	}
	return ip.To4() == nil
}

// GetFullMask returns /32 for an IPv4 address and /128 for an IPv6 address
func GetFullMask(address string) (string, error) {
	if IsIPv4(address) {
		return "/32", nil
	}
	if IsIPv6(address) {
		return "/128", nil
	}
	return "", fmt.Errorf("failed to parse %s as either IPv4 or IPv6", address)
}
