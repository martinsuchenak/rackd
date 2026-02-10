package discovery

import (
	"fmt"
	"net"
)

// MaxSubnetBits is the maximum subnet size allowed for scanning (default /16 = 65536 hosts)
const MaxSubnetBits = 16

var ErrSubnetTooLarge = fmt.Errorf("subnet too large: maximum /%d allowed", 32-MaxSubnetBits)
var ErrScanNotFound = fmt.Errorf("scan not found")
var ErrScanNotRunning = fmt.Errorf("scan is not running or pending")

func countHosts(ipNet *net.IPNet) int {
	ones, bits := ipNet.Mask.Size()
	return 1 << (bits - ones)
}

func expandCIDR(ipNet *net.IPNet) []string {
	var ips []string
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)
	ip = ip.Mask(ipNet.Mask)

	for ipNet.Contains(ip) {
		ips = append(ips, ip.String())
		incrementIP(ip)
	}

	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func getTop100Ports() []int {
	return []int{
		7, 9, 13, 21, 22, 23, 25, 26, 37, 53, 79, 80, 81, 88, 106, 110, 111, 113, 119, 135,
		139, 143, 144, 179, 199, 389, 427, 443, 444, 445, 465, 513, 514, 515, 543, 544, 548,
		554, 587, 631, 646, 873, 990, 993, 995, 1025, 1026, 1027, 1028, 1029, 1110, 1433,
		1720, 1723, 1755, 1900, 2000, 2001, 2049, 2121, 2717, 3000, 3128, 3306, 3389, 3986,
		4899, 5000, 5009, 5051, 5060, 5101, 5190, 5357, 5432, 5631, 5666, 5800, 5900, 6000,
		6001, 6646, 7070, 8000, 8008, 8009, 8080, 8081, 8443, 8888, 9100, 9999, 10000, 32768,
		49152, 49153, 49154, 49155, 49156, 49157,
	}
}
