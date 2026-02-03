# API Rate Limiting

Rackd includes built-in rate limiting to prevent API abuse and ensure fair resource usage across clients.

## Overview

Rate limiting is **disabled by default** and can be enabled via environment variables. When enabled, it tracks request rates per client (by IP address or API key) and returns HTTP 429 (Too Many Requests) when limits are exceeded.

## Configuration

Rate limiting is configured via environment variables:

```bash
# Enable rate limiting (default: false)
RATE_LIMIT_ENABLED=true

# Maximum requests per window (default: 100)
RATE_LIMIT_REQUESTS=100

# Time window for rate limiting (default: 1m)
RATE_LIMIT_WINDOW=1m
```

### Example Configurations

**Strict limits for public API:**
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=60
RATE_LIMIT_WINDOW=1m
```

**Relaxed limits for internal use:**
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=1000
RATE_LIMIT_WINDOW=1m
```

**Burst protection:**
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=10
RATE_LIMIT_WINDOW=10s
```

## How It Works

### Client Identification

Rate limits are applied per client, identified by:

1. **API Key** (if present) - Extracted from `Authorization: Bearer <token>` header
2. **IP Address** (fallback) - Extracted from:
   - `X-Forwarded-For` header (first IP)
   - `X-Real-IP` header
   - `RemoteAddr` (direct connection)

### Token Bucket Algorithm

Rackd uses a token bucket algorithm:

- Each client gets a bucket with `RATE_LIMIT_REQUESTS` tokens
- Each request consumes one token
- Buckets refill completely after `RATE_LIMIT_WINDOW` expires
- Requests are blocked when bucket is empty

### Localhost Bypass

Requests from localhost (`127.0.0.1`, `::1`) **always bypass** rate limiting. This ensures:

- Local development is never rate limited
- CLI commands work without restrictions
- Health checks and monitoring aren't affected

## Response Headers

All API responses include rate limit headers:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 42
X-RateLimit-Reset: 2026-02-03T16:45:00Z
```

When rate limited:

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 2026-02-03T16:45:00Z
Retry-After: 2026-02-03T16:45:00Z
Content-Type: application/json

{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED"
}
```

## Usage Examples

### Checking Rate Limit Status

```bash
curl -i http://localhost:8080/api/devices

HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 2026-02-03T16:45:00Z
```

### Handling Rate Limits

```bash
# Make requests until rate limited
for i in {1..150}; do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/devices
done

# Output:
# 200
# 200
# ...
# 200  (100th request)
# 429  (101st request - rate limited)
# 429
# ...
```

### Using API Keys

```bash
# Different API keys have separate rate limits
curl -H "Authorization: Bearer key1" http://localhost:8080/api/devices
curl -H "Authorization: Bearer key2" http://localhost:8080/api/devices
```

## Best Practices

### For API Consumers

1. **Monitor headers** - Check `X-RateLimit-Remaining` to avoid hitting limits
2. **Implement backoff** - Respect `Retry-After` header when rate limited
3. **Use API keys** - Get dedicated rate limits instead of sharing IP-based limits
4. **Batch operations** - Use bulk endpoints to reduce request count

### For Administrators

1. **Start disabled** - Enable rate limiting only when needed
2. **Monitor usage** - Check logs for rate limit events
3. **Adjust limits** - Tune based on actual usage patterns
4. **Use reverse proxy** - Consider nginx/Caddy rate limiting for additional protection

## Monitoring

Rate limit events are logged at DEBUG level:

```
level=debug msg="Rate limit exceeded" client=192.168.1.100 path=/api/devices
```

To see rate limit logs:

```bash
LOG_LEVEL=debug rackd server
```

## Performance

Rate limiting has minimal overhead:

- **Memory**: ~100 bytes per active client
- **CPU**: Negligible (simple token bucket operations)
- **Cleanup**: Automatic cleanup of inactive clients every 2x window duration

## Limitations

1. **In-memory only** - Rate limits reset on server restart
2. **Single instance** - No coordination across multiple Rackd instances
3. **No per-endpoint limits** - All endpoints share the same limit

For multi-instance deployments, consider using a reverse proxy (nginx, Caddy) or API gateway for distributed rate limiting.

## Troubleshooting

### Rate limiting not working

Check configuration:
```bash
# Verify rate limiting is enabled
curl http://localhost:8080/api/config | jq .rate_limit

# Check server logs for rate limit configuration
rackd server
# Output should include:
# level=info msg="Rate limiting enabled" requests=100 window=1m0s
```

### Localhost being rate limited

This should never happen. If it does, check:
- Server is binding to correct address
- Request is actually coming from localhost
- No proxy is rewriting `RemoteAddr`

### Different rate limits than expected

Rate limits are per-client:
- API key clients get separate limits
- IP-based clients share limits per IP
- Localhost always bypasses limits

## Security Considerations

1. **Not a DDoS solution** - Rate limiting helps but isn't sufficient for DDoS protection
2. **IP spoofing** - Trust `X-Forwarded-For` only behind trusted proxies
3. **API key sharing** - Shared keys share rate limits
4. **Localhost bypass** - Ensure server isn't exposed with localhost access

## See Also

- [Authentication](authentication.md) - API key management
- [Configuration](configuration.md) - Environment variables
- [Monitoring](monitoring.md) - Metrics and observability
- [Security](security.md) - Security best practices
