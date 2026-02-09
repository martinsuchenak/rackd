package discovery

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
)

type SNMPScanner struct {
	credStore credentials.Storage
	timeout   time.Duration
	retries   int
}

func NewSNMPScanner(credStore credentials.Storage, timeout time.Duration) *SNMPScanner {
	return &SNMPScanner{credStore: credStore, timeout: timeout, retries: 2}
}

type SNMPResult struct {
	SysDescr    string
	SysName     string
	SysLocation string
	SysContact  string
	Interfaces  []SNMPInterface
	ARPEntries  []struct {
		IP  string
		MAC string
	}
}

type SNMPInterface struct {
	Index       int
	Description string
	Type        int
	Speed       uint64
	MAC         string
	AdminStatus int
	OperStatus  int
}

func (s *SNMPScanner) Scan(ctx context.Context, ip string, credentialID string) (*SNMPResult, error) {
	cred, err := s.credStore.Get(credentialID)
	if err != nil {
		return nil, fmt.Errorf("credential lookup failed: %w", err)
	}

	client := &gosnmp.GoSNMP{
		Target:  ip,
		Port:    161,
		Timeout: s.timeout,
		Retries: s.retries,
	}

	switch cred.Type {
	case "snmp_v2c":
		// WARNING: SNMPv2c transmits community string in cleartext.
		// Use only on trusted networks. Consider SNMPv3 for production.
		client.Version = gosnmp.Version2c
		client.Community = cred.SNMPCommunity
	case "snmp_v3":
		client.Version = gosnmp.Version3
		client.SecurityModel = gosnmp.UserSecurityModel
		client.MsgFlags = gosnmp.AuthPriv
		client.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 cred.SNMPV3User,
			AuthenticationProtocol:   gosnmp.SHA,
			AuthenticationPassphrase: cred.SNMPV3Auth,
			PrivacyProtocol:          gosnmp.AES,
			PrivacyPassphrase:        cred.SNMPV3Priv,
		}
	default:
		return nil, fmt.Errorf("unsupported credential type for SNMP: %s", cred.Type)
	}

	if err := client.ConnectIPv4(); err != nil {
		return nil, fmt.Errorf("SNMP connect failed: %w", err)
	}
	defer client.Conn.Close()

	result := &SNMPResult{}
	s.getSysInfo(client, result)
	s.getInterfaces(client, result)
	s.getARPTable(client, result)

	return result, nil
}

func (s *SNMPScanner) getSysInfo(client *gosnmp.GoSNMP, result *SNMPResult) {
	oids := []string{
		"1.3.6.1.2.1.1.1.0", // sysDescr
		"1.3.6.1.2.1.1.5.0", // sysName
		"1.3.6.1.2.1.1.6.0", // sysLocation
		"1.3.6.1.2.1.1.4.0", // sysContact
	}
	resp, err := client.Get(oids)
	if err != nil {
		return
	}
	for _, v := range resp.Variables {
		switch v.Name {
		case ".1.3.6.1.2.1.1.1.0":
			result.SysDescr = string(v.Value.([]byte))
		case ".1.3.6.1.2.1.1.5.0":
			result.SysName = string(v.Value.([]byte))
		case ".1.3.6.1.2.1.1.6.0":
			result.SysLocation = string(v.Value.([]byte))
		case ".1.3.6.1.2.1.1.4.0":
			result.SysContact = string(v.Value.([]byte))
		}
	}
}

func (s *SNMPScanner) getInterfaces(client *gosnmp.GoSNMP, result *SNMPResult) {
	ifTable := "1.3.6.1.2.1.2.2.1"

	type ifaceData struct {
		index       int
		description string
		ifType      int
		speed       uint64
		mac         string
		adminStatus int
		operStatus  int
	}

	ifaces := make(map[int]*ifaceData)

	err := client.Walk(ifTable, func(pdu gosnmp.SnmpPDU) error {
		oidParts := strings.Split(pdu.Name, ".")
		if len(oidParts) < 11 {
			return nil
		}

		index, err := strconv.Atoi(oidParts[10])
		if err != nil {
			return nil
		}

		if ifaces[index] == nil {
			ifaces[index] = &ifaceData{index: index}
		}

		switch strings.Join(oidParts[0:10], ".") {
		case ".1.3.6.1.2.1.2.2.1.2":
			if val, ok := pdu.Value.(string); ok {
				ifaces[index].description = val
			}
		case ".1.3.6.1.2.1.2.2.1.3":
			if val, ok := pdu.Value.(int); ok {
				ifaces[index].ifType = val
			}
		case ".1.3.6.1.2.1.2.2.1.5":
			if val, ok := pdu.Value.(uint); ok {
				ifaces[index].speed = uint64(val)
			}
		case ".1.3.6.1.2.1.2.2.1.6":
			if val, ok := pdu.Value.([]byte); ok && len(val) == 6 {
				ifaces[index].mac = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", val[0], val[1], val[2], val[3], val[4], val[5])
			}
		case ".1.3.6.1.2.1.2.2.1.7":
			if val, ok := pdu.Value.(int); ok {
				ifaces[index].adminStatus = val
			}
		case ".1.3.6.1.2.1.2.2.1.8":
			if val, ok := pdu.Value.(int); ok {
				ifaces[index].operStatus = val
			}
		}

		return nil
	})

	if err != nil {
		return
	}

	for _, iface := range ifaces {
		result.Interfaces = append(result.Interfaces, SNMPInterface{
			Index:       iface.index,
			Description: iface.description,
			Type:        iface.ifType,
			Speed:       iface.speed,
			MAC:         iface.mac,
			AdminStatus: iface.adminStatus,
			OperStatus:  iface.operStatus,
		})
	}
}

func (s *SNMPScanner) getARPTable(client *gosnmp.GoSNMP, result *SNMPResult) {
	arpTable := "1.3.6.1.2.1.4.22.1"

	type arpEntry struct {
		ifIndex  int
		physAddr string
		netAddr  string
	}

	arpMap := make(map[int]*arpEntry)

	err := client.Walk(arpTable, func(pdu gosnmp.SnmpPDU) error {
		oidParts := strings.Split(pdu.Name, ".")
		if len(oidParts) < 12 {
			return nil
		}

		ifIndex, err := strconv.Atoi(oidParts[10])
		if err != nil {
			return nil
		}

		if arpMap[ifIndex] == nil {
			arpMap[ifIndex] = &arpEntry{ifIndex: ifIndex}
		}

		switch oidParts[11] {
		case "1":
			if val, ok := pdu.Value.(int); ok {
				arpMap[ifIndex].ifIndex = val
			}
		case "2":
			if val, ok := pdu.Value.([]byte); ok && len(val) == 6 {
				arpMap[ifIndex].physAddr = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", val[0], val[1], val[2], val[3], val[4], val[5])
			}
		case "3":
			if val, ok := pdu.Value.(string); ok {
				arpMap[ifIndex].netAddr = val
			}
		}

		return nil
	})

	if err != nil {
		return
	}

	for _, entry := range arpMap {
		if entry.physAddr != "" && entry.netAddr != "" {
			result.ARPEntries = append(result.ARPEntries, struct {
				IP  string
				MAC string
			}{
				IP:  entry.netAddr,
				MAC: entry.physAddr,
			})
		}
	}
}

func (s *SNMPScanner) IsAvailable(ip string, cred *model.Credential) bool {
	client := &gosnmp.GoSNMP{
		Target:  ip,
		Port:    161,
		Timeout: 2 * time.Second,
		Retries: 1,
	}
	if cred.Type == "snmp_v2c" {
		client.Version = gosnmp.Version2c
		client.Community = cred.SNMPCommunity
	} else {
		return false
	}
	if err := client.ConnectIPv4(); err != nil {
		return false
	}
	defer client.Conn.Close()
	_, err := client.Get([]string{"1.3.6.1.2.1.1.1.0"})
	return err == nil
}
