package client

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func PrintDeviceTable(devices []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tMAKE/MODEL\tOS\tDATACENTER")
	for _, d := range devices {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			getString(d, "id"),
			getString(d, "name"),
			getString(d, "make_model"),
			getString(d, "os"),
			getString(d, "datacenter_id"))
	}
	w.Flush()
}

func PrintNetworkTable(networks []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSUBNET\tVLAN\tDATACENTER")
	for _, n := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n",
			getString(n, "id"),
			getString(n, "name"),
			getString(n, "subnet"),
			n["vlan_id"],
			getString(n, "datacenter_id"))
	}
	w.Flush()
}

func PrintDatacenterTable(datacenters []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tLOCATION")
	for _, dc := range datacenters {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			getString(dc, "id"),
			getString(dc, "name"),
			getString(dc, "location"))
	}
	w.Flush()
}

func PrintDiscoveredTable(devices []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tIP\tHOSTNAME\tSTATUS")
	for _, d := range devices {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			getString(d, "id"),
			getString(d, "ip"),
			getString(d, "hostname"),
			getString(d, "status"))
	}
	w.Flush()
}

func PrintConflictTable(conflicts []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tDESCRIPTION")
	for _, c := range conflicts {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			getString(c, "id"),
			getString(c, "type"),
			getString(c, "status"),
			getString(c, "description"))
	}
	w.Flush()
}

func PrintJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

func PrintYAML(data interface{}) {
	printYAMLValue(data, 0)
}

func printYAMLValue(v interface{}, indent int) {
	prefix := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case map[string]interface{}:
		for k, v := range val {
			fmt.Printf("%s%s:", prefix, k)
			if isScalar(v) {
				fmt.Print(" ")
				printYAMLValue(v, 0)
			} else {
				fmt.Println()
				printYAMLValue(v, indent+1)
			}
		}
	case []interface{}:
		for _, item := range val {
			fmt.Printf("%s- ", prefix)
			printYAMLValue(item, indent+1)
		}
	case string:
		fmt.Printf("%q\n", val)
	case nil:
		fmt.Println("null")
	default:
		fmt.Printf("%v\n", val)
	}
}

func isScalar(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		return false
	}
	return true
}

func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getString(m map[string]interface{}, key string) string {
	return GetString(m, key)
}
