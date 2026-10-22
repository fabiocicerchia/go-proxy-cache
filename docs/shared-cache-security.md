# Shared-cache security

- Requests containing `Authorization`, `Cookie`, `Range`, or a `no-cache` or
  `no-store` request directive bypass shared-cache lookup and storage.
- Responses containing `Set-Cookie`, `private`, `no-cache`, or `no-store` are
  never stored, including when a fallback or negative TTL is configured.
- Collapsed forwarding is off by default. When enabled, it shares only explicit
  `public` 200 responses, without credentials or cookies, and at most 1 MiB.
  Requests must match the domain, transport, URL, Host and complete request
  headers. Other responses are fetched separately for every waiting caller.
- PURGE is denied when both its IP allowlist and secret are empty. Configure
  `server.purge.allowed_ips`, `server.purge.secret`, or both before using PURGE.
  When both are configured, both checks must pass.

After upgrading, clear existing shared-cache entries before serving traffic.
Older entries may contain personalised responses or use the previous Vary key.
