# PMail single-admin wildcard catch-all build

For the complete Chinese Docker, SQLite, configuration-file and field
reference, see [docs/CONFIGURATION_CN.md](docs/CONFIGURATION_CN.md). A valid
JSON template is provided as
[config.admin-catchall.sqlite.example.json](config.admin-catchall.sqlite.example.json).

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
  -p 25:25 -p 80:80 -p 443:443 -p 110:110 \
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
