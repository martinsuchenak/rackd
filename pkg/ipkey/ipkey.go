// Package ipkey stores IP addresses and prefixes so that every database
// backend compares them the same way.
//
// An address is stored as a 16-byte key: IPv6 as is, IPv4 mapped into IPv6
// (::ffff:a.b.c.d). A prefix is stored as two keys, its first and last
// address. Byte-wise comparison of keys then orders addresses numerically,
// and containment, overlap and range scans become plain comparisons:
//
//	address in prefix:    start <= ip AND ip <= end
//	prefix contains p:    start <= p.start AND p.end <= end
//	prefixes overlap:     a.start <= b.end AND b.start <= a.end
//
// SQLite (BLOB), PostgreSQL (BYTEA), MySQL (VARBINARY) and SQL Server
// (VARBINARY) all compare binary values byte by byte, so these queries behave
// the same on each. PostgreSQL's inet type is deliberately not used.
//
// IPv4 keys sort inside ::ffff:0:0/96, so an IPv6 prefix that covers that
// range (::/0, for example) also covers every IPv4 key. Store the family with
// each row, or query within one family, when that matters.
package ipkey

import (
	"errors"
	"fmt"
	"net/netip"
)

// Size is the length of a key in bytes.
const Size = 16

// ErrInvalidKey is returned for a key that isn't Size bytes long.
var ErrInvalidKey = errors.New("ipkey: key must be 16 bytes")

// FromAddr returns the key for an address. IPv4 addresses, including
// IPv4-mapped IPv6 ones, become ::ffff:a.b.c.d; an IPv6 zone is dropped.
func FromAddr(a netip.Addr) []byte {
	b := a.Unmap().As16()
	return b[:]
}

// Parse returns the key for an address in text form.
func Parse(s string) ([]byte, error) {
	a, err := netip.ParseAddr(s)
	if err != nil {
		return nil, err
	}
	return FromAddr(a), nil
}

// ToAddr returns the address a key stores. Keys in ::ffff:0:0/96 come back
// as IPv4 addresses.
func ToAddr(key []byte) (netip.Addr, error) {
	if len(key) != Size {
		return netip.Addr{}, ErrInvalidKey
	}
	return netip.AddrFrom16([Size]byte(key)).Unmap(), nil
}

// Range returns the keys of a prefix's first and last address. Host bits
// set in p are ignored: 10.1.2.3/24 gives the range of 10.1.2.0/24.
func Range(p netip.Prefix) (start, end []byte, err error) {
	if !p.IsValid() {
		return nil, nil, fmt.Errorf("ipkey: invalid prefix %v", p)
	}
	p = p.Masked()
	bits := p.Bits()
	if p.Addr().Is4() {
		bits += 96 // the IPv4 bits sit in the last 32 of the 128
	}
	first := FromAddr(p.Addr())
	last := append([]byte(nil), first...)
	for i := bits; i < 128; i++ {
		last[i/8] |= 0x80 >> (i % 8)
	}
	return first, last, nil
}

// ParsePrefix returns the range keys of a prefix in CIDR form.
func ParsePrefix(s string) (start, end []byte, err error) {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return nil, nil, err
	}
	return Range(p)
}
