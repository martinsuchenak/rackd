package discovery

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

// TestApplySysInfoHostilePDUs verifies that malformed or hostile SNMP response
// variables (nil values, wrong ASN.1 types) do not panic the scanner.
// Regression test: unchecked v.Value.([]byte) assertions previously crashed
// the whole process during a scan.
func TestApplySysInfoHostilePDUs(t *testing.T) {
	cases := []struct {
		name string
		pdus []gosnmp.SnmpPDU
	}{
		{
			name: "NoSuchObject has nil Value",
			pdus: []gosnmp.SnmpPDU{
				{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.NoSuchObject, Value: nil},
				{Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.NoSuchInstance, Value: nil},
			},
		},
		{
			name: "EndOfMibView has nil Value",
			pdus: []gosnmp.SnmpPDU{
				{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.EndOfMibView, Value: nil},
			},
		},
		{
			name: "wrong type Integer instead of OctetString",
			pdus: []gosnmp.SnmpPDU{
				{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.Integer, Value: 42},
				{Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.Counter64, Value: uint64(7)},
			},
		},
		{
			name: "unknown type with nil Value",
			pdus: []gosnmp.SnmpPDU{
				{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.UnknownType, Value: nil},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := &SNMPResult{}
			applySysInfo(tc.pdus, result) // must not panic
			if result.SysDescr != "" || result.SysName != "" {
				t.Errorf("non-octet-string values must be ignored, got SysDescr=%q SysName=%q", result.SysDescr, result.SysName)
			}
		})
	}
}

// TestApplySysInfoValidPDUs verifies normal octet-string values are extracted.
func TestApplySysInfoValidPDUs(t *testing.T) {
	result := &SNMPResult{}
	applySysInfo([]gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: []byte("Linux router 6.1")},
		{Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.OctetString, Value: []byte("core-sw-01")},
		{Name: ".1.3.6.1.2.1.1.6.0", Type: gosnmp.OctetString, Value: []byte("rack 4")},
		{Name: ".1.3.6.1.2.1.1.4.0", Type: gosnmp.OctetString, Value: []byte("ops@example.com")},
	}, result)

	if result.SysDescr != "Linux router 6.1" || result.SysName != "core-sw-01" ||
		result.SysLocation != "rack 4" || result.SysContact != "ops@example.com" {
		t.Errorf("unexpected extraction result: %+v", result)
	}
}
