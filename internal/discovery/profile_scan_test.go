package discovery

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestProfileCredentials(t *testing.T) {
	both := &model.ScanProfile{EnableSNMP: true, EnableSSH: true, SNMPCredentialID: "snmp-p", SSHCredentialID: "ssh-p"}
	off := &model.ScanProfile{SNMPCredentialID: "snmp-p", SSHCredentialID: "ssh-p"}
	for _, c := range []struct {
		name              string
		profile           *model.ScanProfile
		snmpIn, sshIn     string
		wantSNMP, wantSSH string
	}{
		{"profile credentials", both, "", "", "snmp-p", "ssh-p"},
		{"overrides win", both, "snmp-o", "ssh-o", "snmp-o", "ssh-o"},
		{"disabled probes get none", off, "snmp-o", "ssh-o", "", ""},
		{"enabled without credential", &model.ScanProfile{EnableSNMP: true}, "", "", "", ""},
	} {
		snmp, ssh := profileCredentials(c.profile, c.snmpIn, c.sshIn)
		if snmp != c.wantSNMP || ssh != c.wantSSH {
			t.Errorf("%s: got %q/%q, want %q/%q", c.name, snmp, ssh, c.wantSNMP, c.wantSSH)
		}
	}
}

// A host that answered keeps its record when its per-host budget runs out;
// only a cancelled scan drops it.
func TestDiscoverHostKeepsHostWhenBudgetExpires(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer func() { _ = store.Close() }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	opts := &ScanOptions{ScanType: "custom", Profile: &model.ScanProfile{ScanType: "custom", Ports: []int{port}}}
	results := &networkScanResults{}

	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()
	if d := scanner.discoverHostWithOptions(expired, "127.0.0.1", "net-1", opts, time.Second, results); d == nil {
		t.Error("host with an open port was dropped when its time budget ran out")
	} else if len(d.OpenPorts) != 1 || d.OpenPorts[0] != port {
		t.Errorf("open ports %v, want [%d]", d.OpenPorts, port)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if d := scanner.discoverHostWithOptions(cancelled, "127.0.0.1", "net-1", opts, time.Second, results); d != nil {
		t.Error("a cancelled scan should not report the host")
	}
}
