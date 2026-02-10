package discovery

import (
	"strings"
)

type DeviceType string

const (
	DeviceTypeServer      DeviceType = "server"
	DeviceTypeWorkstation DeviceType = "workstation"
	DeviceTypeRouter      DeviceType = "router"
	DeviceTypeSwitch      DeviceType = "switch"
	DeviceTypeFirewall    DeviceType = "firewall"
	DeviceTypePrinter     DeviceType = "printer"
	DeviceTypeIoT         DeviceType = "iot"
	DeviceTypeMobile      DeviceType = "mobile"
	DeviceTypeNAS         DeviceType = "nas"
	DeviceTypeAP          DeviceType = "access_point"
	DeviceTypePhone       DeviceType = "phone"
	DeviceTypeCamera      DeviceType = "camera"
	DeviceTypeUnknown     DeviceType = "unknown"
)

type DeviceTypeClassifier struct {
}

func NewDeviceTypeClassifier() *DeviceTypeClassifier {
	return &DeviceTypeClassifier{}
}

type DeviceInfo struct {
	OS       string
	Vendor   string
	Ports    []int
	Services []ServiceInfo
}

type ServiceInfo struct {
	Port     int
	Protocol string
	Service  string
	Version  string
}

func (c *DeviceTypeClassifier) Classify(device *DeviceInfo) DeviceType {
	if device == nil {
		return DeviceTypeUnknown
	}

	scores := map[DeviceType]float64{
		DeviceTypeServer:      c.scoreServer(device),
		DeviceTypeWorkstation: c.scoreWorkstation(device),
		DeviceTypeRouter:      c.scoreRouter(device),
		DeviceTypeSwitch:      c.scoreSwitch(device),
		DeviceTypeFirewall:    c.scoreFirewall(device),
		DeviceTypePrinter:     c.scorePrinter(device),
		DeviceTypeIoT:         c.scoreIoT(device),
		DeviceTypeMobile:      c.scoreMobile(device),
		DeviceTypeNAS:         c.scoreNAS(device),
		DeviceTypeAP:          c.scoreAP(device),
		DeviceTypePhone:       c.scorePhone(device),
		DeviceTypeCamera:      c.scoreCamera(device),
	}

	var bestType DeviceType = DeviceTypeUnknown
	var bestScore float64 = 0

	for deviceType, score := range scores {
		if score > bestScore {
			bestScore = score
			bestType = deviceType
		}
	}

	if bestScore > 0 {
		return bestType
	}

	return DeviceTypeUnknown
}

func (c *DeviceTypeClassifier) scoreServer(device *DeviceInfo) float64 {
	score := 0.0

	serverPorts := []int{22, 80, 443, 8080, 3306, 5432, 6379, 27017, 5672, 9200, 9090}
	for _, port := range device.Ports {
		for _, sp := range serverPorts {
			if port == sp {
				score += 0.2
				break
			}
		}
	}

	if strings.Contains(strings.ToLower(device.OS), "linux") ||
		strings.Contains(strings.ToLower(device.OS), "unix") {
		score += 0.3
	}

	if strings.Contains(strings.ToLower(device.OS), "windows server") {
		score += 0.5
	}

	serverVendors := []string{"Dell", "HPE", "Lenovo", "Supermicro"}
	for _, vendor := range serverVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.2
			break
		}
	}

	for _, svc := range device.Services {
		if strings.Contains(strings.ToLower(svc.Service), "http") ||
			strings.Contains(strings.ToLower(svc.Service), "ssh") ||
			strings.Contains(strings.ToLower(svc.Service), "database") {
			score += 0.15
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreWorkstation(device *DeviceInfo) float64 {
	score := 0.0

	if strings.Contains(strings.ToLower(device.OS), "windows") &&
		!strings.Contains(strings.ToLower(device.OS), "server") {
		score += 0.4
	}

	if strings.Contains(strings.ToLower(device.OS), "macos") {
		score += 0.4
	}

	if strings.Contains(strings.ToLower(device.OS), "ubuntu desktop") {
		score += 0.3
	}

	workstationVendors := []string{"Apple", "Dell", "HP", "Lenovo", "Acer", "Asus"}
	for _, vendor := range workstationVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.2
			break
		}
	}

	if c.hasPort(device, 3389) {
		score += 0.2
	}

	if c.hasPort(device, 5900) {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreRouter(device *DeviceInfo) float64 {
	score := 0.0

	routerVendors := []string{"Cisco", "Juniper", "Fortinet", "Ubiquiti", "Mikrotik", "TP-Link", "Netgear", "Linksys", "Asus"}
	for _, vendor := range routerVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	if strings.Contains(strings.ToLower(device.OS), "router") ||
		strings.Contains(strings.ToLower(device.OS), "ios") ||
		strings.Contains(strings.ToLower(device.OS), "junos") {
		score += 0.3
	}

	routerPorts := []int{23, 22, 161, 179}
	for _, port := range device.Ports {
		for _, rp := range routerPorts {
			if port == rp {
				score += 0.15
				break
			}
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreSwitch(device *DeviceInfo) float64 {
	score := 0.0

	switchVendors := []string{"Cisco", "Juniper", "Arista", "HPE", "Dell", "Brocade", "Ubiquiti"}
	for _, vendor := range switchVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.6
			break
		}
	}

	if strings.Contains(strings.ToLower(device.OS), "switch") ||
		strings.Contains(strings.ToLower(device.OS), "ios") {
		score += 0.3
	}

	if c.hasPort(device, 161) {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreFirewall(device *DeviceInfo) float64 {
	score := 0.0

	firewallVendors := []string{"Palo Alto", "Fortinet", "Cisco", "Checkpoint", "Sophos", "Juniper"}
	for _, vendor := range firewallVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	if strings.Contains(strings.ToLower(device.OS), "firewall") ||
		strings.Contains(strings.ToLower(device.OS), "palo alto") ||
		strings.Contains(strings.ToLower(device.OS), "fortigate") {
		score += 0.4
	}

	if c.hasPort(device, 443) && strings.Contains(strings.ToLower(device.Vendor), "palo") {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scorePrinter(device *DeviceInfo) float64 {
	score := 0.0

	printerVendors := []string{"HP", "Canon", "Epson", "Brother", "Xerox", "Kyocera", "Ricoh", "Lexmark"}
	for _, vendor := range printerVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	printerPorts := []int{9100, 515, 631}
	for _, port := range device.Ports {
		for _, pp := range printerPorts {
			if port == pp {
				score += 0.2
				break
			}
		}
	}

	for _, svc := range device.Services {
		if strings.Contains(strings.ToLower(svc.Service), "printer") ||
			strings.Contains(strings.ToLower(svc.Service), "ipp") {
			score += 0.2
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreIoT(device *DeviceInfo) float64 {
	score := 0.0

	iotVendors := []string{"Philips Hue", "Nest", "Ring", "Wyze", "TP-Link", "Ecobee"}
	for _, vendor := range iotVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	if strings.Contains(strings.ToLower(device.OS), "iot") ||
		strings.Contains(strings.ToLower(device.OS), "embedded") {
		score += 0.4
	}

	if len(device.Ports) <= 3 && len(device.Ports) > 0 {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreMobile(device *DeviceInfo) float64 {
	score := 0.0

	if strings.Contains(strings.ToLower(device.OS), "ios") ||
		strings.Contains(strings.ToLower(device.OS), "android") {
		score += 0.4
	}

	if strings.Contains(strings.ToLower(device.Vendor), "apple") ||
		strings.Contains(strings.ToLower(device.Vendor), "samsung") ||
		strings.Contains(strings.ToLower(device.Vendor), "google") ||
		strings.Contains(strings.ToLower(device.Vendor), "xiaomi") {
		score += 0.3
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreNAS(device *DeviceInfo) float64 {
	score := 0.0

	nasVendors := []string{"Synology", "QNAP", "Netgear", "Asustor", "WD"}
	for _, vendor := range nasVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	nasPorts := []int{80, 443, 5000, 5001, 8443}
	for _, port := range device.Ports {
		for _, np := range nasPorts {
			if port == np {
				score += 0.15
				break
			}
		}
	}

	for _, svc := range device.Services {
		if strings.Contains(strings.ToLower(svc.Service), "nas") ||
			strings.Contains(strings.ToLower(svc.Service), "storage") {
			score += 0.3
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreAP(device *DeviceInfo) float64 {
	score := 0.0

	apVendors := []string{"Ubiquiti", "Ruckus", "Cisco", "Aruba", "TP-Link", "Netgear", "Meraki"}
	for _, vendor := range apVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	if strings.Contains(strings.ToLower(device.Vendor), "unifi") {
		score += 0.3
	}

	if c.hasPort(device, 8080) && strings.Contains(strings.ToLower(device.Vendor), "ubiquiti") {
		score += 0.2
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scorePhone(device *DeviceInfo) float64 {
	score := 0.0

	phoneVendors := []string{"Cisco", "Polycom", "Yealink", "Grandstream", "Avaya", "Snom"}
	for _, vendor := range phoneVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	phonePorts := []int{5060, 5061}
	for _, port := range device.Ports {
		for _, pp := range phonePorts {
			if port == pp {
				score += 0.3
				break
			}
		}
	}

	for _, svc := range device.Services {
		if strings.Contains(strings.ToLower(svc.Service), "sip") {
			score += 0.2
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) scoreCamera(device *DeviceInfo) float64 {
	score := 0.0

	cameraVendors := []string{"Axis", "Hikvision", "Dahua", "Amcrest", "Reolink", "Foscam"}
	for _, vendor := range cameraVendors {
		if strings.Contains(strings.ToLower(device.Vendor), strings.ToLower(vendor)) {
			score += 0.5
			break
		}
	}

	cameraPorts := []int{80, 554, 8000, 8080}
	for _, port := range device.Ports {
		for _, cp := range cameraPorts {
			if port == cp {
				score += 0.1
				break
			}
		}
	}

	for _, svc := range device.Services {
		if strings.Contains(strings.ToLower(svc.Service), "rtsp") ||
			strings.Contains(strings.ToLower(svc.Service), "camera") {
			score += 0.3
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (c *DeviceTypeClassifier) hasPort(device *DeviceInfo, port int) bool {
	for _, p := range device.Ports {
		if p == port {
			return true
		}
	}
	return false
}
