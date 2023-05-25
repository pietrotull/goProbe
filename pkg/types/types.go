package types

import (
	"errors"
	"net"
	"net/netip"
	"strings"
)

// IPVersion denotes the IP layer version (if any) of a conditional node
type IPVersion int

const (
	IPVersionNone IPVersion = iota // IPVersionNone : Not an IP related node
	IPVersionBoth                  // IPVersionBoth : Node contains subnodes with both IP layer versions

	IPVersionV4 // IPVersionV4 : IPv4 related node
	IPVersionV6 // IPVersionV6 : IPv6 related node
)

// Merge combines two IPVersion instances
func (v IPVersion) Merge(v2 IPVersion) IPVersion {

	if v == IPVersionNone || v2 == IPVersionBoth {
		return v2
	}
	if v2 == IPVersionNone || v == IPVersionBoth {
		return v
	}

	if v != v2 {
		return IPVersionBoth
	}

	return v
}

// IsLimited returns if the IP layer version is limited (i.e. not none or both)
func (v IPVersion) IsLimited() bool {
	return v >= IPVersionV4
}

type Status string

const (
	StatusError Status = "error"
	StatusEmpty Status = "empty"
	StatusOK    Status = "ok"
)

const DefaultTimeOutputFormat = "2006-01-02 15:04:05"

type LabelSelector struct {
	Timestamp bool
	Iface     bool
	Hostname  bool
	HostID    bool
}

type Width = int

const (
	IPv6Width  Width = 16
	IPv4Width  Width = 4
	DPortWidth Width = 2
	ProtoWidth Width = 1

	TimestampWidth Width = 8
)

const (
	sipPos       = 0
	dipPosIPv4   = IPv4Width
	dipPosIPv6   = IPv6Width
	dportPosIPv4 = sipDipIPv4Width
	dportPosIPv6 = sipDipIPv6Width
	protoPosIPv4 = dportPosIPv4 + DPortWidth
	protoPosIPv6 = dportPosIPv6 + DPortWidth

	nonIPKeysWidth  = DPortWidth + ProtoWidth
	sipDipIPv4Width = 2 * IPv4Width
	sipDipIPv6Width = 2 * IPv6Width

	KeyWidthIPv4 = sipDipIPv4Width + nonIPKeysWidth
	KeyWidthIPv6 = sipDipIPv6Width + nonIPKeysWidth
)

// RawIPToAddr converts an ip byte slice to an actual netip.Addr
func RawIPToAddr(ip []byte) netip.Addr {
	zeros := numZeros(ip)
	ind := len(ip)
	if zeros == 12 {
		// only read first 4 bytes (IPv4)
		ind = 4
	}
	netIP, ok := netip.AddrFromSlice(ip[:ind])
	if !ok {
		return netip.Addr{}
	}
	return netIP
}

// RawIPToString converts an ip byte slice to string
func RawIPToString(ip []byte) string {
	return RawIPToAddr(ip).String()
}

func numZeros(ip []byte) uint8 {
	var numZeros uint8

	// count zeros in order to determine whether the address
	// is IPv4 or IPv6
	for i := 4; i < len(ip); i++ {
		if (ip[i] & 0xFF) == 0x00 {
			numZeros++
		}
	}
	return numZeros
}

// IPStringToBytes creates a goDB compatible bytes slice from an IP address string
func IPStringToBytes(ip string) ([]byte, bool, error) {
	var isIPv4 = strings.Contains(ip, ".")

	ipaddr := net.ParseIP(ip)
	if len(ipaddr) == 0 {
		return nil, false, errors.New("IP parse: incorrect format")
	}
	if isIPv4 {
		return []byte{ipaddr[12], ipaddr[13], ipaddr[14], ipaddr[15]}, true, nil
	}
	return ipaddr, false, nil
}
