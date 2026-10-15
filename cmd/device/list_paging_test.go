package device

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

// The API returns 100 devices per request by default; the CLI must page
// through instead of stopping at the first page.
func TestListDevicesFetchesEveryPage(t *testing.T) {
	const total = 2500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		page := []map[string]string{}
		for i := offset; i < offset+limit && i < total; i++ {
			page = append(page, map[string]string{"id": fmt.Sprint(i)})
		}
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer srv.Close()
	c := client.NewClient(&client.Config{ServerURL: srv.URL})

	all, err := listDevices(c, url.Values{}, 0, 0)
	if err != nil || len(all) != total {
		t.Fatalf("all: %d devices, %v; want %d", len(all), err, total)
	}
	some, err := listDevices(c, url.Values{}, 1500, 10)
	if err != nil || len(some) != 1500 || some[0]["id"] != "10" {
		t.Fatalf("limit 1500 offset 10: %d devices starting %v, %v", len(some), some[0]["id"], err)
	}
}
