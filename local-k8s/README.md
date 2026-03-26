# Local Kubernetes Test Environment for Gotenberg

Reproduces the production Kubernetes setup (5 pods, Nginx Ingress circuit-breaker)
locally so you can observe the `--libreoffice-reject-when-busy` behaviour and confirm
connection resets are eliminated.

---

## Prerequisites

| Tool | Notes |
|------|-------|
| **Docker Desktop** with Kubernetes enabled | Settings → Kubernetes → Enable Kubernetes |
| **kubectl** | Comes with Docker Desktop or `brew install kubectl` |
| **Python 3.10+** | `brew install python` |
| **requests library** | `pip install requests` |

---

## Step 1 — Build the Gotenberg image

Run once, and again whenever you change Go code:

```bash
# From the repo root
make build
```

This produces `gotenberg/gotenberg:snapshot` in your local Docker registry.
Docker Desktop's Kubernetes shares the same Docker daemon so the image is immediately
available to pods — no push required.

---

## Step 2 — Install the Nginx Ingress controller

Run once per cluster lifetime:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.12.1/deploy/static/provider/cloud/deploy.yaml

# Wait for the controller to be ready (~30s)
kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx

# Raise the body size limit for large DOCX files
kubectl patch configmap ingress-nginx-controller -n ingress-nginx \
  --type merge \
  -p '{"data":{"proxy-body-size":"200m","client-max-body-size":"200m"}}'

# Expose Nginx on port 3080 (port 80 is typically taken by local dev servers)
kubectl patch svc ingress-nginx-controller -n ingress-nginx \
  --type='json' \
  -p='[{"op":"replace","path":"/spec/ports/0/port","value":3080}]'
```

Verify:

```bash
curl http://localhost:3080/healthz    # should return 200
```

---

## Step 3 — Deploy Gotenberg

```bash
# From the repo root
kubectl apply -f local-k8s/namespace.yaml
kubectl apply -f local-k8s/configmap.yaml
kubectl apply -f local-k8s/deployment.yaml
kubectl apply -f local-k8s/service.yaml
kubectl apply -f local-k8s/ingress.yaml

# Wait for all 5 pods to be Ready
kubectl rollout status deployment/gotenberg -n gotenberg-local
```

Verify the full stack:

```bash
curl http://localhost:3080/health
# {"status":"up","details":{"chromium":{"status":"up"},"libreoffice":{"status":"up"}}}
```

---

## Step 4 — Run the test script

The test script continuously converts every document in a directory using a pool of
concurrent workers. It retries automatically on 429 (pod busy) with exponential backoff.

```bash
cd test-scripts

!!!!!You need to copy some DOCX files into your test-scripts directory to run the test!!!!!!!

# Continuous load — loops through all docs forever, 5 workers
python3 convert_test.py --dir ./

# Single pass through the directory then stop
python3 convert_test.py --dir ./ --once

# Reduce concurrency if you see ENOBUFS / No buffer space errors (see macOS note below)
python3 convert_test.py --dir ./ --concurrency 3

# Save converted PDFs to ./output/
python3 convert_test.py --dir ./ --concurrency 5 --save-pdfs

# Target a specific file only
python3 convert_test.py --dir ./ --file test.docx --concurrency 5
```

### macOS: socket buffer errors with large files

If you see `OSError(55, 'No buffer space available')` when uploading large files
concurrently, raise the macOS TCP buffer limits (run once per boot):

```bash
sudo sysctl -w kern.ipc.maxsockbuf=8388608
sudo sysctl -w net.inet.tcp.sendspace=1048576
sudo sysctl -w net.inet.tcp.recvspace=1048576
```

Or reduce `--concurrency` to 3.

### Test script options

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | `http://localhost:3080` | Gotenberg URL via Nginx |
| `--dir` | `.` | Directory of documents to convert |
| `--file` | *(all supported files)* | Restrict to one file |
| `--concurrency` | `5` | Number of parallel workers |
| `--once` | off | Stop after one pass through the directory |
| `--timeout` | `120` | Read timeout per request (seconds) |
| `--retry-max` | `20` | Max 429 retries before giving up |
| `--retry-delay` | `0.5` | Initial backoff in seconds (doubles each retry, cap 10s) |
| `--save-pdfs` | off | Write output PDFs to `./output/` |

---

## Watching the cluster in real time

Open these in separate terminals while the test script runs:

```bash
# Pod readiness — watch pods as conversions run
kubectl get pods -n gotenberg-local -w

# All pod logs interleaved
kubectl logs -n gotenberg-local -l app=gotenberg -f --max-log-requests=10

# Nginx access log — shows upstream pod IPs and 429 retry hops
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller -f

# Prometheus metrics on a specific pod
POD=$(kubectl get pods -n gotenberg-local --no-headers -o custom-columns=":metadata.name" | head -1)
kubectl exec -n gotenberg-local $POD -- \
  curl -s http://localhost:3000/prometheus/metrics | grep gotenberg_libreoffice
```

---

## How the circuit-breaker works (HTTP layer, not TCP)

**Why not use the readiness probe for queue/CPU detection?**

When the CNI plugin removes a pod from the Endpoints list (readiness failure), it sends
TCP RST packets to every active connection on that pod. If the probe fires because the
pod is busy converting, the RST kills the very conversion that triggered it — this is
the root cause of the `ConnectionResetError(104)` errors seen in production.

**The fix — reject at the HTTP layer instead:**

| Component | Role |
|-----------|------|
| `--libreoffice-reject-when-busy=true` | Gotenberg returns HTTP 429 **before reading the request body** when a conversion is already active |
| `proxy_next_upstream http_429 non_idempotent` | Nginx retries the full request on the next pod — the body is already buffered so no re-upload from the client |
| Readiness probe checks `/health` only | Pods stay in rotation during conversions — no RST packets ever sent |

The busy pod's active conversion is never interrupted. New requests that land on a busy
pod are transparently rerouted to an idle pod at the HTTP layer before any file is
processed.

---

## Re-deploying after code changes

```bash
make build
kubectl rollout restart deployment/gotenberg -n gotenberg-local
kubectl rollout status deployment/gotenberg -n gotenberg-local
```

---

## Teardown

```bash
# Remove Gotenberg workload only (keeps cluster and Nginx)
kubectl delete namespace gotenberg-local

# Full teardown including Nginx Ingress
kubectl delete namespace gotenberg-local
kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.12.1/deploy/static/provider/cloud/deploy.yaml
```
