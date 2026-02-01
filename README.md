# dev-proxy

A simple DNS proxy that redirects specific domains to a local IP address. All other queries are forwarded to Cloudflare (1.1.1.1).

## Usage

```bash
./dev-proxy <local-ip> <domain>
```

## Example

```bash
./dev-proxy 192.168.2.132 myapp.local.
```

This will resolve `*.myapp.local` to `192.168.2.132`.