# Mikan MOD: External JSON Nodes

This patch adds external client-side Xray outbounds to the existing Mikan panel. It does not
create another panel and does not pretend that a third-party VPN server is a managed Mikan
node. Imported servers are delivered through the existing subscription handler:

- Xray-family clients such as Happ receive ordinary share links in the normal Mikan
  subscription response.
- Clash-family clients receive mihomo proxy entries in the normal profile.
- Imported proxies are members of the normal main selector. The `Auto` url-test group
  includes all native Mikan proxies and imported proxies whose `Auto` switch is enabled. If
  every imported proxy is excluded and there are no native proxies, Auto falls back to DIRECT.
- External servers are not counted by Mikan's node traffic accounting and are not subject
  to Mikan node/tariff inbound controls. Use them only where your service terms permit it.

## Supported import

Paste a full Xray JSON document. The importer extracts supported entries from `outbounds`
and ignores local `inbounds`, `dns`, `routing`, `policy`, `metrics`, `freedom`, `blackhole`,
`dns` and unsupported outbound protocols.

Supported protocols: VLESS, VMess, Trojan and Shadowsocks.

Supported transports: TCP, WebSocket, gRPC, HTTPUpgrade and XHTTP. TLS and REALITY fields
are mapped to share-link and mihomo client settings. Unsupported transport/protocol entries
are skipped; an import with no usable proxy is rejected.

The source JSON is stored in PostgreSQL for reference. It contains credentials (UUIDs,
passwords, public keys); protect database backups accordingly. The import endpoint accepts up
to 2 MiB. The panel does not fetch remote subscription URLs or periodically replace pasted
JSON: a raw JSON paste is a snapshot, so update it by importing the provider's new JSON.

## Admin UI

A new **External nodes / Внешние ноды** section appears in the existing panel sidebar.

- Paste JSON and optionally set a common custom name.
- Imported entries are enabled by default and added to the standard subscription.
- Toggle each entry's `Auto` membership independently.
- Disable or delete entries without changing native Mikan nodes.

## API

All routes are under the panel's secret admin path, `/api/v1`:

- `GET /external-nodes` — list imported servers.
- `POST /external-nodes/import` — body `{ "json": "<full Xray JSON>", "name": "optional name" }`.
- `PATCH /external-nodes/{id}` — body may contain `name`, `enabled`, `auto`.
- `DELETE /external-nodes/{id}` — remove an imported server.

Mutating endpoints require an admin session and CSRF token; they are not exposed to support
operators. The panel applies the `0053_external_nodes` PostgreSQL migration on startup.

## Build

Use the Go and Node/pnpm versions pinned by this repository. Build the frontend and Go panel
using the repository's normal release/build procedure. The current execution environment did
not have network access to download the Go 1.27 toolchain, so tests and a release binary could
not be run here.
