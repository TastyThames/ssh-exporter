# ssh-exporter

# ssh-agentless-exporter

Agentless Prometheus exporter (Go) that connects to target hosts via SSH and exposes OS-level metrics over HTTP.

## Quick start (dev)

```bash
go run ./cmd/ssh-exporter
curl http://127.0.0.1:9222/health
```

##Inventory

`deploy/targets.yaml` (human)

`deploy/file_sd/targets.json` (generated for Prometheus file_sd)
