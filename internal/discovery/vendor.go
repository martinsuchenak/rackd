package discovery

import (
	"strings"
	"sync"
)

type OUIDatabase struct {
	entries map[string]string
	mu      sync.RWMutex
}

func NewOUIDatabase() *OUIDatabase {
	db := &OUIDatabase{
		entries: make(map[string]string),
	}
	db.loadCommonOUIs()
	return db
}

func (db *OUIDatabase) loadCommonOUIs() {
	// Common vendor OUIs (simplified)
	commonOUIs := map[string]string{
		"00:00:0c": "Cisco",
		"00:0c:29": "VMware",
		"00:05:85": "Broadcom",
		"00:0b:cd": "3Com",
		"00:0d:b9": "ZyXEL",
		"00:0e:c6": "Cisco",
		"00:0f:ea": "Netgear",
		"00:10:18": "Broadcom",
		"00:10:db": "Dell",
		"00:11:43": "Cisco",
		"00:11:95": "Broadcom",
		"00:12:3f": "Intel Corporate",
		"00:13:20": "Cisco",
		"00:14:22": "Dell",
		"00:14:5e": "Dell",
		"00:14:a5": "Dell",
		"00:14:bf": "Dell",
		"00:15:17": "Hewlett Packard",
		"00:15:58": "Hewlett Packard",
		"00:15:60": "Hewlett Packard",
		"00:15:99": "Hewlett Packard",
		"00:16:35": "Hewlett Packard",
		"00:16:76": "Hewlett Packard",
		"00:16:cb": "Hewlett Packard",
		"00:16:ec": "Hewlett Packard",
		"00:17:08": "Hewlett Packard",
		"00:17:a4": "Hewlett Packard",
		"00:17:c4": "Hewlett Packard",
		"00:18:82": "Hewlett Packard",
		"00:18:8b": "Hewlett Packard",
		"00:19:bb": "Hewlett Packard",
		"00:19:e0": "Hewlett Packard",
		"00:19:e3": "Hewlett Packard",
		"00:1a:4b": "Hewlett Packard",
		"00:1b:78": "Hewlett Packard",
		"00:1b:9e": "Hewlett Packard",
		"00:1c:23": "Hewlett Packard",
		"00:1c:25": "Hewlett Packard",
		"00:1c:2f": "Hewlett Packard",
		"00:1c:42": "Hewlett Packard",
		"00:1c:7f": "Hewlett Packard",
		"00:1c:c4": "Hewlett Packard",
		"00:1d:09": "Hewlett Packard",
		"00:1d:a2": "Hewlett Packard",
		"00:1d:e0": "Hewlett Packard",
		"00:1e:c9": "Hewlett Packard",
		"00:1f:29": "Hewlett Packard",
		"00:21:28": "Hewlett Packard",
		"00:21:70": "Hewlett Packard",
		"00:21:86": "Hewlett Packard",
		"00:21:9b": "Hewlett Packard",
		"00:22:19": "Hewlett Packard",
		"00:22:64": "Hewlett Packard",
		"00:22:75": "Hewlett Packard",
		"00:22:99": "Hewlett Packard",
		"00:23:7d": "Hewlett Packard",
		"00:23:fd": "Hewlett Packard",
		"00:24:81": "Hewlett Packard",
		"00:24:e8": "Hewlett Packard",
		"00:25:64": "Hewlett Packard",
		"00:25:90": "Hewlett Packard",
		"00:25:b3": "Hewlett Packard",
		"00:26:55": "Hewlett Packard",
		"00:26:9a": "Hewlett Packard",
		"00:23:ae": "Dell",
		"00:26:b9": "Hewlett Packard",
		"00:50:56": "VMware",
		"00:60:08": "Apple",
		"00:0a:95": "Apple",
		"ac:87:a3": "Apple",
		"b4:2e:99": "Apple",
		"dc:a6:32": "Apple",
		"e4:b3:18": "Apple",
		"f0:18:98": "Apple",
		"54:83:3a": "Apple",
		"28:cf:e9": "Apple",
		"bc:d1:d3": "Apple",
		"3c:15:c2": "Apple",
		"a4:d1:d2": "Apple",
		"88:e9:fe": "Apple",
		"c4:2c:03": "Apple",
		"ec:ee:fb": "Apple",
		"e0:ac:cb": "Apple",
		"9c:b6:d0": "Apple",
		"f8:ff:c2": "Apple",
		"a8:20:66": "Apple",
		"d4:9a:20": "Apple",
		"70:73:cb": "Apple",
		"9c:93:4e": "Apple",
		"cc:b2:55": "Apple",
		"00:e0:4c": "Realtek",
		"00:1a:a0": "Realtek",
		"00:1e:ec": "Realtek",
		"00:22:b0": "Realtek",
		"00:24:01": "Realtek",
		"00:30:67": "Realtek",
		"00:26:9e": "Intel Corporate",
		"00:1b:21": "Intel Corporate",
		"00:1b:38": "Intel Corporate",
		"00:1d:d8": "Intel Corporate",
		"00:a0:c9": "Intel Corporate",
		"3c:d9:2b": "Intel Corporate",
		"60:67:20": "Intel Corporate",
		"68:05:ca": "Intel Corporate",
		"a0:36:9f": "Intel Corporate",
		"bc:5f:f4": "Intel Corporate",
		"f4:8e:38": "Intel Corporate",
		"00:04:ac": "Dell",
		"00:04:76": "Dell",
		"00:08:74": "Dell",
		"00:0b:db": "Dell",
		"00:13:72": "Dell",
		"00:14:85": "Dell",
		"00:15:c5": "Dell",
		"00:22:fa": "Dell",
		"00:60:b0": "Dell",
		"00:60:97": "Dell",
		"00:00:00": "Unknown",
		"ff:ff:ff": "Broadcast",
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	for oui, vendor := range commonOUIs {
		db.entries[oui] = vendor
	}
}

func (db *OUIDatabase) Lookup(mac string) string {
	if len(mac) < 8 {
		return ""
	}

	oui := strings.ToLower(mac[:8])

	db.mu.RLock()
	defer db.mu.RUnlock()

	if vendor, ok := db.entries[oui]; ok {
		return vendor
	}

	return ""
}

func (db *OUIDatabase) AddEntry(oui, vendor string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.entries[oui] = vendor
}

func (db *OUIDatabase) Count() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.entries)
}
