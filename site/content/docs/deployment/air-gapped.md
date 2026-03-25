---
title: "Air-Gapped Environments"
weight: 3
---


Some environments have no outbound internet access — no Docker Hub, no GitHub
Releases, no PyPI. This page covers the adjustments needed to deploy `olake-tui`
and its infrastructure in fully air-gapped networks.

> **Tracking issue:** [datazip-inc/olake-ui#340](https://github.com/datazip-inc/olake-ui/issues/340)
> covers air-gapped support across the OLake ecosystem.

## Constraints

In an air-gapped environment you typically cannot:

- Pull container images from Docker Hub or GitHub Container Registry
- Download `olake-tui` binaries from GitHub Releases
- Fetch `releases.json` for version/update checks
- Run `helm repo add` against public chart repositories

All artifacts must be pre-staged on internal mirrors or transferred manually.

## Container Images

### Mirror to Internal Registry

Pre-pull and push the required images to your internal registry:

```bash
# On a machine with internet access
IMAGES=(
  "postgres:13"
  "elasticsearch:7.17.10"
  "temporalio/auto-setup:1.22.3"
  "temporalio/ui:2.16.2"
  "olakego/ui-worker:latest"
)

INTERNAL=registry.internal.example.com

for img in "${IMAGES[@]}"; do
  docker pull "$img"
  docker tag "$img" "$INTERNAL/$img"
  docker push "$INTERNAL/$img"
done
```

### Point Docker Compose at Internal Registry

The `docker-compose.yml` respects the `CONTAINER_REGISTRY_BASE` variable:

```bash
export CONTAINER_REGISTRY_BASE=registry.internal.example.com
docker compose up -d
```

All image references in the compose file are prefixed with this value, so no file
edits are needed.

## olake-tui Binary

Download the binary on a connected machine and transfer it via USB, SCP, or your
artifact store:

```bash
# On connected machine
curl -LO https://github.com/datazip-inc/olake-tui/releases/download/v0.2.0/olake-tui-linux-amd64
chmod +x olake-tui-linux-amd64

# Transfer to air-gapped host
scp olake-tui-linux-amd64 air-gapped-host:/usr/local/bin/olake-tui
```

## Release URL and Version Checks

`olake-tui` can check for available connector versions by fetching a `releases.json`
file. In air-gapped environments you have two options:

### Option A — Internal Mirror

Host `releases.json` on an internal HTTP server and point `olake-tui` at it:

```bash
olake-tui --release-url https://artifacts.internal.example.com/olake/releases.json \
          --db-url "..." --temporal-host "..."
```

Or via environment variable:

```bash
export OLAKE_RELEASE_URL=https://artifacts.internal.example.com/olake/releases.json
```

The JSON format matches the public releases endpoint. Copy it from a connected
machine periodically:

```bash
curl -o releases.json https://api.github.com/repos/datazip-inc/olake/releases
```

### Option B — Omit (Fallback Behavior)

If `--release-url` is omitted **and** the `OLAKE_RELEASE_URL` environment variable
is unset, `olake-tui` skips version fetching entirely. The version selector in the
TUI will show only the versions already known from previous runs or manually
configured connectors. No network calls are made.

This is the safest option for strict air-gapped environments where even internal
HTTP mirrors are restricted.

## Helm Charts (Kubernetes)

For Kubernetes deployments, package the Helm charts on a connected machine:

```bash
helm repo add olake https://charts.olake.io
helm repo update
helm pull olake/olake --destination ./charts/
helm pull olake/olake-tui --destination ./charts/
```

Transfer the `.tgz` files to the air-gapped cluster and install from local paths:

```bash
helm install olake ./charts/olake-*.tgz \
  -f values-olake-override.yaml \
  --namespace olake --create-namespace

helm install olake-tui ./charts/olake-tui-*.tgz \
  --namespace olake \
  --set db.url="postgres://temporal:temporal@olake-postgresql:5432/temporal?sslmode=disable" \
  --set temporal.host="olake-temporal:7233" \
  --set image.registry="registry.internal.example.com"
```

## Verification Checklist

After deploying in an air-gapped environment, verify:

- [ ] All containers are running (`docker compose ps` or `kubectl get pods`)
- [ ] `olake-tui` starts without network errors
- [ ] Connector discovery works (sources/destinations appear)
- [ ] A test sync job completes end-to-end
- [ ] If using `--release-url`, version list populates correctly

## Keeping Up to Date

Establish a periodic process to:

1. Pull updated container images on a connected machine
2. Push them to the internal registry
3. Download new `olake-tui` binaries
4. Update `releases.json` if using Option A
5. Restart services or roll out new Kubernetes deployments
