# Performance Testing

Rackd includes a load tool, `tools/loadtest`, that seeds a server with test data and measures API latency and throughput under concurrent clients. Results depend on hardware, disk, network and data, so the project does not publish figures: run the tool against your own deployment.

## What It Does

By default the tool seeds:

- 1 datacenter.
- 50 `/20` networks, each with one pool covering the whole range.
- 10,000 devices with 10 IP addresses each: 100,000 addresses, about half of each network.

All seeded names start with `lt-`.

It then runs 50 concurrent clients for a 10-second warm-up and a 60-second measured period. Each client sends its next request as soon as the previous one returns, so the server is never idle. Requests are picked at random with these weights:

| Operation | Share | Endpoint |
| --- | ---: | --- |
| Get a device | 25% | `GET /api/v1/devices/{id}` |
| Update a device | 15% | `PUT /api/v1/devices/{id}` |
| List a page of 100 devices | 10% | `GET /api/v1/devices?limit=100&offset=N` |
| Search devices by name | 10% | `GET /api/v1/devices/search?q=...` |
| Global search by IP | 10% | `GET /api/v1/search?q=10.x.x.x` |
| Next free IP in a pool | 10% | `GET /api/v1/pools/{id}/next-ip` |
| List devices in a network | 5% | `GET /api/v1/devices?network_id=...` |
| Network utilization | 5% | `GET /api/v1/networks/{id}/utilization` |
| Dashboard | 5% | `GET /api/v1/dashboard` |
| Create and delete a device | 5% | `POST /api/v1/devices`, then `DELETE` |

The report lists, for each operation, the operations per second, the p50, p95, p99 and maximum latency, and the error count. It also shows the server's heap in use before and after the run, read from `/metrics`.

## Running It

Run the tool against a disposable server: it writes test data and creates an API key named `loadtest-<timestamp>`.

The bulk endpoints use the login rate limit's settings, and the general rate limiter would throttle 50 clients, so start the server with both relaxed. A fresh data directory has no users, so also create the first admin:

```bash
RATE_LIMIT_ENABLED=false LOGIN_RATE_LIMIT_REQUESTS=100000 \
  INITIAL_ADMIN_USERNAME=admin INITIAL_ADMIN_PASSWORD='<admin password>' \
  rackd server --data-dir /tmp/rackd-loadtest
```

Build the tool from a checkout:

```bash
go build -o loadtest ./tools/loadtest
```

Run it with an admin account's password, which it uses once to create the API key:

```bash
RACKD_LOADTEST_PASSWORD='<admin password>' ./loadtest -url http://127.0.0.1:8080 -out results.md
```

To use an existing API key instead of signing in, pass it with `-token '<api key>'`.

| Flag | Default | Meaning |
| --- | --- | --- |
| `-url` | `http://127.0.0.1:8080` | Server URL |
| `-user` | `admin` | User that creates the API key |
| `-token` | | Use this API key instead of logging in |
| `-seed` | `true` | Seed the data first; `-seed=false` reuses data from an earlier run |
| `-devices` | `10000` | Devices to seed |
| `-ips-per-device` | `10` | Addresses per device |
| `-networks` | `50` | `/20` networks to spread the addresses over |
| `-clients` | `50` | Concurrent clients |
| `-duration` | `1m` | Measured run time |
| `-warmup` | `10s` | Run time before measuring starts |
| `-out` | | Also write the results table to this file |

## Reading the Results

- Look at p95 and p99 rather than p50. They show what a busy user waits for.
- Errors should be zero. The first error for each operation is printed under the table.
- Run the tool on a different machine from the server where you can. When both share a machine, they compete for CPU and the numbers read low.
- To compare two builds, run both on the same machine against freshly seeded data.
- SQLite runs one write at a time, so write-heavy operations slow down first as clients are added.

## Cleaning Up

Delete the server's data directory. On a server you keep, revoke the `loadtest-<timestamp>` API key and delete the devices, networks and datacenter whose names start with `lt-`.
