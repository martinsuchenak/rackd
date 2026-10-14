// Command loadtest seeds a rackd server with networks, pools and devices, then
// measures API latency and throughput under concurrent load.
//
// See docs/performance.md for how the published numbers were produced.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type config struct {
	url          string
	user, pass   string
	token        string
	seed         bool
	devices      int
	ipsPerDevice int
	networks     int
	clients      int
	duration     time.Duration
	warmup       time.Duration
	out          string
}

func main() {
	var cfg config
	flag.StringVar(&cfg.url, "url", "http://127.0.0.1:8080", "rackd server URL")
	flag.StringVar(&cfg.user, "user", "admin", "username used to create an API key")
	flag.StringVar(&cfg.token, "token", "", "API key to use instead of logging in")
	flag.BoolVar(&cfg.seed, "seed", true, "create the test data before the run")
	flag.IntVar(&cfg.devices, "devices", 10000, "devices to seed")
	flag.IntVar(&cfg.ipsPerDevice, "ips-per-device", 10, "IP addresses per seeded device")
	flag.IntVar(&cfg.networks, "networks", 50, "/20 networks to spread the addresses over")
	flag.IntVar(&cfg.clients, "clients", 50, "concurrent API clients")
	flag.DurationVar(&cfg.duration, "duration", time.Minute, "measured run time")
	flag.DurationVar(&cfg.warmup, "warmup", 10*time.Second, "run time before measuring starts")
	flag.StringVar(&cfg.out, "out", "", "also write the results table to this file")
	flag.Parse()
	cfg.pass = os.Getenv("RACKD_LOADTEST_PASSWORD")

	if err := run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, "loadtest:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) error {
	switch {
	case cfg.networks < 1 || cfg.networks > 1024:
		return errors.New("-networks must be between 1 and 1024")
	case cfg.devices < 1 || cfg.ipsPerDevice < 1 || cfg.clients < 1:
		return errors.New("-devices, -ips-per-device and -clients must be at least 1")
	case cfg.duration <= 0 || cfg.warmup < 0:
		return errors.New("-duration must be positive and -warmup not negative")
	}
	perNetwork := (cfg.devices + cfg.networks - 1) / cfg.networks * cfg.ipsPerDevice
	if perNetwork > hostsPerNetwork-hostsReserved {
		return fmt.Errorf("%d addresses per /20 network do not fit; raise -networks", perNetwork)
	}

	c := newClient(cfg.url, cfg.clients)
	if cfg.token == "" {
		if cfg.pass == "" {
			return errors.New("set RACKD_LOADTEST_PASSWORD, or pass -token")
		}
		token, err := c.createAPIKey(ctx, cfg.user, cfg.pass)
		if err != nil {
			return err
		}
		cfg.token = token
	}
	c.token = cfg.token

	var data *dataset
	var err error
	if cfg.seed {
		start := time.Now()
		if data, err = seed(ctx, c, cfg); err != nil {
			return fmt.Errorf("seed: %w (seed into an empty server, or pass -seed=false to reuse data from an earlier run)", err)
		}
		fmt.Printf("Seeded %d devices with %d addresses in %d networks in %s\n",
			len(data.deviceIDs), len(data.deviceIDs)*cfg.ipsPerDevice, len(data.networkIDs), time.Since(start).Round(time.Second))
	} else if data, err = discover(ctx, c); err != nil {
		return fmt.Errorf("discover: %w", err)
	}
	if len(data.deviceIDs) == 0 || len(data.networkIDs) == 0 {
		return errors.New("no load-test devices or networks found; run with -seed")
	}

	memBefore := c.serverMemory(ctx)
	results := load(ctx, c, data, cfg)
	memAfter := c.serverMemory(ctx)

	report := formatReport(cfg, results, memBefore, memAfter)
	fmt.Print(report)
	if cfg.out != "" {
		return os.WriteFile(cfg.out, []byte(report), 0o644)
	}
	return nil
}

// ── HTTP client ──────────────────────────────────────────────────────────────

type client struct {
	base  string
	token string
	http  *http.Client
}

func newClient(base string, clients int) *client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = clients * 2
	transport.MaxIdleConnsPerHost = clients * 2
	return &client{
		base: strings.TrimRight(base, "/"),
		http: &http.Client{Transport: transport, Timeout: 60 * time.Second},
	}
}

// do sends a request and decodes a JSON response into out (when non-nil). It
// returns the status code; a non-2xx status is also returned as an error.
func (c *client) do(ctx context.Context, method, path string, body, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return resp.StatusCode, fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, bytes.TrimSpace(msg))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, nil
	}
	return resp.StatusCode, json.NewDecoder(resp.Body).Decode(out)
}

// doRetry retries requests rejected with 429, for seeding through the
// login-rate-limited bulk endpoints.
func (c *client) doRetry(ctx context.Context, method, path string, body, out any) error {
	for attempt := 0; ; attempt++ {
		status, err := c.do(ctx, method, path, body, out)
		if status != http.StatusTooManyRequests || attempt == 20 {
			return err
		}
		if attempt == 0 {
			fmt.Fprintln(os.Stderr, "rate limited; retrying (raise LOGIN_RATE_LIMIT_REQUESTS and RATE_LIMIT_REQUESTS on the server to seed faster)")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// createAPIKey logs in with a session and creates an API key for the run.
func (c *client) createAPIKey(ctx context.Context, user, pass string) (string, error) {
	jar, _ := cookiejar.New(nil)
	session := &client{base: c.base, http: &http.Client{Jar: jar, Timeout: c.http.Timeout}}
	login := map[string]string{"username": user, "password": pass}
	if _, err := session.do(ctx, http.MethodPost, "/api/v1/auth/login", login, nil); err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	var key struct {
		Key string `json:"key"`
	}
	req := map[string]string{"name": "loadtest-" + time.Now().UTC().Format("20060102-150405"), "description": "tools/loadtest"}
	// Session-authenticated writes need the CSRF header.
	session.http.Transport = headerTransport{"X-Requested-With": "XMLHttpRequest"}
	if _, err := session.do(ctx, http.MethodPost, "/api/v1/keys", req, &key); err != nil {
		return "", fmt.Errorf("create API key: %w", err)
	}
	return key.Key, nil
}

type headerTransport map[string]string

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range h {
		r.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(r)
}

// serverMemory reads go_memory_alloc_bytes from /metrics, or -1.
func (c *client) serverMemory(ctx context.Context) int64 {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/metrics", nil)
	if err != nil {
		return -1
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return -1
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	for _, line := range strings.Split(string(body), "\n") {
		if v, ok := strings.CutPrefix(line, "go_memory_alloc_bytes "); ok {
			n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err == nil {
				return n
			}
		}
	}
	return -1
}

// ── Test data ────────────────────────────────────────────────────────────────

const (
	namePrefix      = "lt-"
	hostsPerNetwork = 4094 // usable hosts in a /20
	hostsReserved   = 94   // left free so next-ip always finds an address
	bulkBatch       = 100  // the bulk endpoint's limit
)

type dataset struct {
	deviceIDs  []string
	networkIDs []string
	poolIDs    []string
	networks   []netip.Prefix
}

// networkPrefix returns the n-th /20 starting at 10.0.0.0.
func networkPrefix(n int) netip.Prefix {
	base := uint32(10)<<24 | uint32(n)<<12
	return netip.PrefixFrom(netip.AddrFrom4([4]byte{byte(base >> 24), byte(base >> 16), byte(base >> 8), byte(base)}), 20)
}

// host returns the i-th host address (1-based) in p.
func host(p netip.Prefix, i int) netip.Addr {
	a := p.Addr().As4()
	v := uint32(a[0])<<24 | uint32(a[1])<<16 | uint32(a[2])<<8 | uint32(a[3])
	v += uint32(i)
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func seed(ctx context.Context, c *client, cfg config) (*dataset, error) {
	var dc struct {
		ID string `json:"id"`
	}
	if err := c.doRetry(ctx, http.MethodPost, "/api/v1/datacenters", map[string]string{"name": namePrefix + "dc"}, &dc); err != nil {
		return nil, err
	}

	data := &dataset{}
	for n := range cfg.networks {
		prefix := networkPrefix(n)
		var network, pool struct {
			ID string `json:"id"`
		}
		body := map[string]any{"name": fmt.Sprintf("%snet-%03d", namePrefix, n), "subnet": prefix.String(), "datacenter_id": dc.ID}
		if err := c.doRetry(ctx, http.MethodPost, "/api/v1/networks", body, &network); err != nil {
			return nil, err
		}
		body = map[string]any{"name": fmt.Sprintf("%spool-%03d", namePrefix, n), "start_ip": host(prefix, 1).String(), "end_ip": host(prefix, hostsPerNetwork).String()}
		if err := c.doRetry(ctx, http.MethodPost, "/api/v1/networks/"+network.ID+"/pools", body, &pool); err != nil {
			return nil, err
		}
		data.networkIDs = append(data.networkIDs, network.ID)
		data.poolIDs = append(data.poolIDs, pool.ID)
		data.networks = append(data.networks, prefix)
	}

	for start := 0; start < cfg.devices; start += bulkBatch {
		batch := make([]map[string]any, 0, bulkBatch)
		for i := start; i < min(start+bulkBatch, cfg.devices); i++ {
			n := i % cfg.networks
			addresses := make([]map[string]any, cfg.ipsPerDevice)
			for j := range addresses {
				addresses[j] = map[string]any{
					"ip":         host(data.networks[n], (i/cfg.networks)*cfg.ipsPerDevice+j+1).String(),
					"type":       "ipv4",
					"label":      fmt.Sprintf("eth%d", j),
					"network_id": data.networkIDs[n],
					"pool_id":    data.poolIDs[n],
				}
			}
			batch = append(batch, map[string]any{
				"name":          fmt.Sprintf("%sdev-%05d", namePrefix, i),
				"hostname":      fmt.Sprintf("%sdev-%05d.example.test", namePrefix, i),
				"description":   "load test device",
				"make_model":    "Generic 1U",
				"os":            "linux",
				"datacenter_id": dc.ID,
				"status":        "active",
				"tags":          []string{"loadtest", fmt.Sprintf("rack-%03d", i/40)},
				"addresses":     addresses,
			})
		}
		var result struct {
			Failed int      `json:"failed"`
			Errors []string `json:"errors"`
		}
		if err := c.doRetry(ctx, http.MethodPost, "/api/v1/devices/bulk", batch, &result); err != nil {
			return nil, err
		}
		if result.Failed > 0 {
			return nil, fmt.Errorf("bulk create: %d failed: %v", result.Failed, result.Errors)
		}
	}

	ids, err := listDeviceIDs(ctx, c)
	if err != nil {
		return nil, err
	}
	data.deviceIDs = ids
	return data, nil
}

// discover finds load-test data seeded by an earlier run.
func discover(ctx context.Context, c *client) (*dataset, error) {
	var networks []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Subnet string `json:"subnet"`
	}
	if _, err := c.do(ctx, http.MethodGet, "/api/v1/networks?limit=1000", nil, &networks); err != nil {
		return nil, err
	}
	data := &dataset{}
	for _, n := range networks {
		if !strings.HasPrefix(n.Name, namePrefix) {
			continue
		}
		var pools []struct {
			ID string `json:"id"`
		}
		if _, err := c.do(ctx, http.MethodGet, "/api/v1/networks/"+n.ID+"/pools", nil, &pools); err != nil {
			return nil, err
		}
		prefix, err := netip.ParsePrefix(n.Subnet)
		if err != nil || len(pools) == 0 {
			continue
		}
		data.networkIDs = append(data.networkIDs, n.ID)
		data.poolIDs = append(data.poolIDs, pools[0].ID)
		data.networks = append(data.networks, prefix)
	}
	ids, err := listDeviceIDs(ctx, c)
	if err != nil {
		return nil, err
	}
	data.deviceIDs = ids
	return data, nil
}

func listDeviceIDs(ctx context.Context, c *client) ([]string, error) {
	var ids []string
	for offset := 0; ; offset += 1000 {
		var page []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if _, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/devices?limit=1000&offset=%d", offset), nil, &page); err != nil {
			return nil, err
		}
		for _, d := range page {
			if strings.HasPrefix(d.Name, namePrefix+"dev-") {
				ids = append(ids, d.ID)
			}
		}
		if len(page) < 1000 {
			return ids, nil
		}
	}
}

// ── Workload ─────────────────────────────────────────────────────────────────

type operation struct {
	name   string
	weight int
	run    func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error
}

// operations is the request mix: mostly reads, as in day-to-day IPAM use, with
// a steady stream of writes. Weights are percentages. Each operation is one
// request, except "create + delete device", which is two.
var operations = []operation{
	{"get device", 25, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		_, err := c.do(ctx, http.MethodGet, "/api/v1/devices/"+pick(r, d.deviceIDs), nil, nil)
		return err
	}},
	{"list devices (page of 100)", 10, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		offset := r.IntN(max(1, len(d.deviceIDs)-100))
		_, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/devices?limit=100&offset=%d", offset), nil, nil)
		return err
	}},
	{"list devices in a network", 5, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		_, err := c.do(ctx, http.MethodGet, "/api/v1/devices?limit=100&network_id="+pick(r, d.networkIDs), nil, nil)
		return err
	}},
	{"search devices by name", 10, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		_, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/devices/search?q=%sdev-%04d", namePrefix, r.IntN(1000)), nil, nil)
		return err
	}},
	{"global search by IP", 10, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		n := r.IntN(len(d.networks))
		_, err := c.do(ctx, http.MethodGet, "/api/v1/search?q="+host(d.networks[n], 1+r.IntN(2000)).String(), nil, nil)
		return err
	}},
	{"network utilization", 5, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		_, err := c.do(ctx, http.MethodGet, "/api/v1/networks/"+pick(r, d.networkIDs)+"/utilization", nil, nil)
		return err
	}},
	{"next free IP in a pool", 10, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		_, err := c.do(ctx, http.MethodGet, "/api/v1/pools/"+pick(r, d.poolIDs)+"/next-ip", nil, nil)
		return err
	}},
	{"dashboard", 5, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		_, err := c.do(ctx, http.MethodGet, "/api/v1/dashboard", nil, nil)
		return err
	}},
	{"update device", 15, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		body := map[string]string{"description": fmt.Sprintf("updated %d", r.Int64())}
		_, err := c.do(ctx, http.MethodPut, "/api/v1/devices/"+pick(r, d.deviceIDs), body, nil)
		return err
	}},
	{"create + delete device", 5, func(ctx context.Context, c *client, d *dataset, r *rand.Rand) error {
		var created struct {
			ID string `json:"id"`
		}
		body := map[string]any{"name": fmt.Sprintf("%stmp-%d", namePrefix, r.Int64()), "status": "planned", "tags": []string{"loadtest-tmp"}}
		if _, err := c.do(ctx, http.MethodPost, "/api/v1/devices", body, &created); err != nil {
			return err
		}
		_, err := c.do(ctx, http.MethodDelete, "/api/v1/devices/"+created.ID, nil, nil)
		return err
	}},
}

func pick(r *rand.Rand, ids []string) string { return ids[r.IntN(len(ids))] }

type sample struct {
	op      int
	latency time.Duration
	err     bool
}

type opResult struct {
	name      string
	latencies []time.Duration
	errors    int
	firstErr  string
}

func load(ctx context.Context, c *client, d *dataset, cfg config) []opResult {
	total := 0
	for _, op := range operations {
		total += op.weight
	}
	measureFrom := time.Now().Add(cfg.warmup)
	deadline := measureFrom.Add(cfg.duration)
	fmt.Printf("Running %d clients for %s (after %s warm-up)...\n", cfg.clients, cfg.duration, cfg.warmup)

	results := make([]opResult, len(operations))
	for i, op := range operations {
		results[i].name = op.name
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for w := range cfg.clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := rand.New(rand.NewPCG(uint64(w), uint64(time.Now().UnixNano())))
			var local []sample
			firstErrs := map[int]string{}
			for {
				now := time.Now()
				if now.After(deadline) {
					break
				}
				n := r.IntN(total)
				i := 0
				for ; n >= operations[i].weight; i++ {
					n -= operations[i].weight
				}
				start := time.Now()
				err := operations[i].run(ctx, c, d, r)
				if start.Before(measureFrom) {
					continue
				}
				local = append(local, sample{op: i, latency: time.Since(start), err: err != nil})
				if _, seen := firstErrs[i]; err != nil && !seen {
					firstErrs[i] = err.Error()
				}
			}
			mu.Lock()
			defer mu.Unlock()
			for _, s := range local {
				res := &results[s.op]
				if s.err {
					res.errors++
					continue
				}
				res.latencies = append(res.latencies, s.latency)
			}
			for i, e := range firstErrs {
				if results[i].firstErr == "" {
					results[i].firstErr = e
				}
			}
		}()
	}
	wg.Wait()
	return results
}

// ── Report ───────────────────────────────────────────────────────────────────

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(float64(len(sorted)-1) * p)
	return sorted[i]
}

func ms(d time.Duration) string { return fmt.Sprintf("%.1f", float64(d)/float64(time.Millisecond)) }

func formatReport(cfg config, results []opResult, memBefore, memAfter int64) string {
	var b strings.Builder
	secs := cfg.duration.Seconds()
	var all []time.Duration
	errors := 0
	fmt.Fprintf(&b, "\n%d clients, %s measured after %s warm-up, %d devices × %d IPs in %d networks.\n",
		cfg.clients, cfg.duration, cfg.warmup, cfg.devices, cfg.ipsPerDevice, cfg.networks)
	fmt.Fprintf(&b, "Load generator: %s/%s, %d CPUs, %s.\n\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version())
	fmt.Fprintln(&b, "| Operation | Ops/s | p50 (ms) | p95 (ms) | p99 (ms) | Max (ms) | Errors |")
	fmt.Fprintln(&b, "| --- | ---: | ---: | ---: | ---: | ---: | ---: |")
	for _, res := range results {
		slices.Sort(res.latencies)
		all = append(all, res.latencies...)
		errors += res.errors
		maxLatency := time.Duration(0)
		if len(res.latencies) > 0 {
			maxLatency = res.latencies[len(res.latencies)-1]
		}
		fmt.Fprintf(&b, "| %s | %.0f | %s | %s | %s | %s | %d |\n", res.name, float64(len(res.latencies))/secs,
			ms(percentile(res.latencies, 0.50)), ms(percentile(res.latencies, 0.95)), ms(percentile(res.latencies, 0.99)), ms(maxLatency), res.errors)
	}
	slices.Sort(all)
	fmt.Fprintf(&b, "| **All** | **%.0f** | **%s** | **%s** | **%s** | **%s** | **%d** |\n\n", float64(len(all))/secs,
		ms(percentile(all, 0.50)), ms(percentile(all, 0.95)), ms(percentile(all, 0.99)), ms(percentile(all, 1)), errors)
	if memBefore >= 0 && memAfter >= 0 {
		fmt.Fprintf(&b, "Server heap in use: %.0f MB before, %.0f MB after.\n", float64(memBefore)/1e6, float64(memAfter)/1e6)
	}
	for _, res := range results {
		if res.firstErr != "" {
			fmt.Fprintf(&b, "First error (%s): %s\n", res.name, res.firstErr)
		}
	}
	return b.String()
}
