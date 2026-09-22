# PMail single-admin wildcard catch-all build

For the complete Chinese Docker, SQLite, configuration-file and field
reference, see [docs/CONFIGURATION_CN.md](docs/CONFIGURATION_CN.md). A valid
JSON template is provided as
[config.admin-catchall.sqlite.example.json](config.admin-catchall.sqlite.example.json).

For the source/security review, compatibility changes and remaining limitations,
see [the audit report](docs/AUDIT_2026-09-21_CN.md). This review is not a production
deployment certificate or a claim of complete IMAP/POP3 interoperability.

This fork keeps PMail's existing user and mailbox model and adds a narrowly
scoped mode for one administrator:

- every value in `domains` is a root domain;
- when `acceptSubdomains` is true, every valid descendant of a configured root
  is local;
- unknown local-parts are delivered to the configured administrator in
  `catchAllAccount`;
- unauthenticated SMTP rejects non-local recipients during `RCPT TO`;
- authenticated senders are limited to configured roots and their descendants;
- DKIM selects the matched configured root as `d=`;
- `tlsNames` decouples certificate identifiers from all recipient domains;
- `outboundHostname` supplies a stable EHLO name that can match PTR.

## First-time setup

Use `admin` as the administrator login. In PMail's setup UI:

- enter one of the owned roots as the primary SMTP domain;
- enter the single canonical service hostname (for example,
  `mail.example-mail-server.com`) as the web domain;
- add the other root domains in the multi-domain list.

This fork writes `acceptSubdomains=true`, `catchAllAccount=admin`, uses the web
domain as `outboundHostname`, and requests TLS only for that same hostname. The
resulting mounted `config/config.json` should contain values like:

```json
{
  "domain": "117799.xyz",
  "domains": [
    "117799.xyz",
    "example.com",
    "another-example.net"
  ],
  "acceptSubdomains": true,
  "catchAllAccount": "admin",
  "outboundHostname": "mail.example-mail-server.com",
  "tlsNames": ["mail.example-mail-server.com"]
}
```

The administrator account named by `catchAllAccount` must exist, be enabled,
and have `is_admin=1`. Existing installations can add these fields manually and
restart PMail.

## DNS records shown by first-time setup

The setup wizard uses `outboundHostname` as the one canonical mail service
host. For every configured root it now shows:

```text
@                   MX   mail.example-mail-server.com
*                   MX   mail.example-mail-server.com
@                   TXT  v=spf1 mx ~all
*                   TXT  v=spf1 mx ~all
default._domainkey  TXT  v=DKIM1; k=rsa; p=...
_dmarc              TXT  v=DMARC1; p=none; sp=none; adkim=r; aspf=r
```

It shows one A record for the canonical host instead of creating `smtp`,
`imap`, and `pop` hosts under every recipient root. The table explicitly shows
MX priority `10`; enter it in the DNS provider's separate priority field when
the provider uses one. The MX target must
resolve to A/AAAA and must not be a CNAME.

DNS wildcards are synthesized only when the queried name does not already
exist. If `shop.example.com` has any explicit DNS record, add exact MX/SPF
records at `shop` and branch wildcard records at `*.shop` to cover descendants.
The setup page calls this out explicitly. Existing installations must publish
the same records manually for every root later added to `domains`, then restart
PMail; only root domains, never `*.example.com`, belong in `config.json`.

`tlsNames` prevents the built-in ACME client from deriving `smtp.`, `pop.`, and
`imap.` names for every recipient domain. All MX records and mail clients should
use the single hostname in `tlsNames`; it must resolve to the VPS and its
certificate must cover that hostname.

## Build locally

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v2.9.7-admincatchall.1 \
  --build-arg GITHASH="$(git rev-parse HEAD)" \
  --file DockerfileGithubAction \
  --tag ghcr.io/YOUR-GITHUB-USER/pmail:v2.9.7-admincatchall.1 \
  --push .
```

Run `make build_fe` before that command because `DockerfileGithubAction` copies
the already built web assets from `server/listen/http_server/dist`.

## Publish with GitHub Actions

When this workflow exists only on the customization branch, GitHub may not
offer it in the manual workflow list. Push a new unused `0.*` version tag
pointing to this branch instead (for example `git tag 0.07 && git push fork 0.07`).
This starts the catch-all image workflow from that tag, publishes only its
versioned image, and leaves `latest` unchanged. Never move an already published
tag. Do not also publish a GitHub Release for the same tag unless you intend
to run the separate release workflows and update `latest`.

If the workflow is available in GitHub's manual workflow list:

1. Push this source to a GitHub repository under your account.
2. Open **Actions → Build admin catch-all image → Run workflow**.
3. Keep the tag `v2.9.7-admincatchall.1` or choose another valid OCI tag.
4. Make the resulting GHCR package public if anonymous `docker pull` is needed.

The pull command will be:

```bash
docker pull ghcr.io/YOUR-GITHUB-USER/YOUR-REPOSITORY:v2.9.7-admincatchall.1
```

The workflow uses the lower-case GitHub `owner/repository` automatically and
publishes `linux/386`, `linux/amd64`, `linux/arm/v7`, and `linux/arm64` images.
Only the upstream owner can publish under `ghcr.io/jinnrry/pmail`; a fork is
published under the fork owner's namespace.

Alternatively, set the image once and start the included Compose file:

```bash
export PMAIL_IMAGE=ghcr.io/YOUR-GITHUB-USER/YOUR-REPOSITORY:v2.9.7-admincatchall.1
docker compose -f docker-compose.admin-catchall.yml up -d
```

## Runtime example

```bash
docker run -d --name pmail --restart unless-stopped \
  -p 25:25 -p 80:80 -p 443:443 \
  -p 465:465 -p 587:587 -p 995:995 -p 993:993 \
  -v "$(pwd)/config:/work/config" \
  ghcr.io/YOUR-GITHUB-USER/YOUR-REPOSITORY:v2.9.7-admincatchall.1
```

`/work` is the final image's `WORKDIR`, not a required source directory on the
host. The image contains `/work/pmail`, and PMail consequently reads runtime
state from `/work/config`. The source-tree directory `server/config` is used
while compiling the image. Mount only runtime data, not Go source files.

Before exposing port 25, verify that external recipients receive SMTP 550 at
the RCPT stage and configured root/subdomain recipients receive SMTP 250.
