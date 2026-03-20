---
title: "Kubernetes (Helm)"
weight: 2
---

# Kubernetes Deployment

For production workloads, deploy OLake infrastructure using the official
**olake-helm** chart, then layer on `olake-tui` with its own Helm chart.

> **Detailed walkthrough:** [Deploying OLake on Kubernetes with Helm](https://olake.io/blog/deploying-olake-on-kubernetes-helm)

## Prerequisites

- Kubernetes 1.24+ cluster with `kubectl` configured
- Helm 3.12+
- `olake-tui` binary on your local machine

## Step 1 — Add the OLake Helm Repository

```bash
helm repo add olake https://charts.olake.io
helm repo update
```

## Step 2 — Install OLake Infrastructure

The official `olake` chart deploys PostgreSQL, Temporal, Elasticsearch, and the
OLake Worker. Because `olake-tui` replaces the BFF layer, create a values override
file to disable it:

```yaml
# values-olake-override.yaml
bff:
  enabled: false

signup-init:
  enabled: false

worker:
  image:
    pullPolicy: Always
  env:
    OLAKE_CALLBACK_URL: ""
```

Install the chart:

```bash
helm install olake olake/olake \
  -f values-olake-override.yaml \
  --namespace olake \
  --create-namespace
```

Verify all pods are running:

```bash
kubectl get pods -n olake
```

Expected output:

```
NAME                              READY   STATUS    RESTARTS   AGE
olake-postgresql-0                1/1     Running   0          2m
olake-elasticsearch-0             1/1     Running   0          2m
olake-temporal-0                  1/1     Running   0          2m
olake-worker-5d8f9c7b6-xk2pq     1/1     Running   0          2m
```

## Step 3 — Install olake-tui Chart

The `olake-tui` Helm chart runs database migrations as a Kubernetes **Job** and
optionally creates a port-forwarding-friendly Service.

```bash
helm install olake-tui olake/olake-tui \
  --namespace olake \
  --set db.url="postgres://temporal:temporal@olake-postgresql:5432/temporal?sslmode=disable" \
  --set temporal.host="olake-temporal:7233" \
  --set migrate.adminUser="admin" \
  --set migrate.adminPassword="changeme"
```

This creates:

1. A **Job** (`olake-tui-migrate`) that runs `olake-tui --migrate-only` to create
   tables and seed the admin user.
2. A **ConfigMap** with connection details for local TUI usage.

Check the migration Job completed:

```bash
kubectl get jobs -n olake
```

```
NAME                  COMPLETIONS   DURATION   AGE
olake-tui-migrate     1/1           8s         30s
```

## Step 4 — Connect the Local TUI

`olake-tui` is a terminal application — it runs on your workstation and connects
to the in-cluster services via `kubectl port-forward`.

Open two port-forward sessions:

```bash
# Terminal 1 — PostgreSQL
kubectl port-forward svc/olake-postgresql 5432:5432 -n olake

# Terminal 2 — Temporal
kubectl port-forward svc/olake-temporal 7233:7233 -n olake
```

Then start the TUI:

```bash
olake-tui \
  --db-url "postgres://temporal:temporal@localhost:5432/temporal?sslmode=disable" \
  --temporal-host localhost:7233
```

No `--migrate` needed — the Helm Job already ran migrations.

## Upgrading

```bash
helm repo update
helm upgrade olake olake/olake -f values-olake-override.yaml -n olake
helm upgrade olake-tui olake/olake-tui -n olake
```

The upgrade re-runs the migration Job to apply any schema changes.

## Uninstalling

```bash
helm uninstall olake-tui -n olake
helm uninstall olake -n olake
kubectl delete namespace olake
```

## Helm Values Reference

See [Configuration Reference]({{< relref "configuration" >}}) for the full
`olake-tui` chart `values.yaml` documentation, including all configurable fields
for database, Temporal, migration, and image settings.
