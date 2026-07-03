#!/usr/bin/env bash
# user_data.sh — EC2 bootstrap script for Gotenberg
# Terraform templatefile() variables (substituted before the script runs):
#   aws_region        — AWS region (e.g. us-east-1)
#   ecr_registry      — <account_id>.dkr.ecr.<region>.amazonaws.com
#   ecr_repository    — ECR repository name (gotenberg)
#   gotenberg_version — image tag to pull
#   secret_name       — Secrets Manager secret name
#
# Any other $${...} in this file uses double-dollar so Terraform leaves them
# alone; bash then evaluates them as normal variables at runtime.

set -euxo pipefail
exec > >(tee /var/log/user-data.log | logger -t user-data -s 2>/dev/console) 2>&1

# ── [1/7] System update ──────────────────────────────────────────────────────
echo "==> [1/7] System update"
dnf update -y

# ── [2/7] Install and start Docker ──────────────────────────────────────────
echo "==> [2/7] Install Docker"
dnf install -y docker
systemctl enable docker
systemctl start docker

# ── [3/7] Install Docker Compose plugin ─────────────────────────────────────
# AL2023 does not ship docker-compose-plugin via dnf; install the v2 binary directly.
echo "==> [3/7] Install docker compose v2 binary"
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL "https://github.com/docker/compose/releases/download/v2.27.0/docker-compose-linux-x86_64" \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
docker compose version

# ── [4/7] Authenticate with ECR ─────────────────────────────────────────────
echo "==> [4/7] ECR login"
aws ecr get-login-password --region "${aws_region}" \
  | docker login --username AWS --password-stdin "${ecr_registry}"

# ── [5/7] Fetch Basic Auth password from Secrets Manager ────────────────────
echo "==> [5/7] Fetch secret"
BASIC_AUTH_PASSWORD=$(aws secretsmanager get-secret-value \
  --region "${aws_region}" \
  --secret-id "${secret_name}" \
  --query SecretString \
  --output text)

# ── [6/7] Write .env file ────────────────────────────────────────────────────
echo "==> [6/7] Write /opt/gotenberg/.env"
mkdir -p /opt/gotenberg

# NOTE: $${VAR} below → Terraform leaves them alone; bash expands at runtime.
cat > /opt/gotenberg/.env <<EOF
# Registry
DOCKER_REGISTRY=${ecr_registry}
DOCKER_REPOSITORY=${ecr_repository}
GOTENBERG_VERSION=${gotenberg_version}

# API
API_PORT=3000
API_BIND_IP=0.0.0.0
API_PORT_FROM_ENV=false
API_START_TIMEOUT=30s
API_TIMEOUT=30s
API_BODY_LIMIT=64MiB
API_ROOT_PATH=/
API_CORRELATION_ID_HEADER=Correlation-Id
API_ENABLE_BASIC_AUTH=true
API_DOWNLOAD_FROM_ALLOW_LIST=
API_DOWNLOAD_FROM_DENY_LIST=
API_DOWNLOAD_FROM_MAX_RETRY=4
API_DISABLE_DOWNLOAD_FROM=false
API_DISABLE_HEALTH_CHECK_ROUTE_TELEMETRY=false
API_DISABLE_ROOT_ROUTE_TELEMETRY=false
API_DISABLE_DEBUG_ROUTE_TELEMETRY=false
API_DISABLE_VERSION_ROUTE_TELEMETRY=false
API_ENABLE_DEBUG_ROUTE=false

# Basic Auth
GOTENBERG_API_BASIC_AUTH_USERNAME=q10-firma
GOTENBERG_API_BASIC_AUTH_PASSWORD=$${BASIC_AUTH_PASSWORD}

# Chromium
CHROMIUM_RESTART_AFTER=10
CHROMIUM_AUTO_START=false
CHROMIUM_MAX_QUEUE_SIZE=0
CHROMIUM_IDLE_SHUTDOWN_TIMEOUT=30s
CHROMIUM_MAX_CONCURRENCY=5
CHROMIUM_START_TIMEOUT=20s
CHROMIUM_ALLOW_INSECURE_LOCALHOST=false
CHROMIUM_IGNORE_CERTIFICATE_ERRORS=false
CHROMIUM_DISABLE_WEB_SECURITY=false
CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES=false
CHROMIUM_HOST_RESOLVER_RULES=
CHROMIUM_PROXY_SERVER=
CHROMIUM_ALLOW_LIST=
CHROMIUM_DENY_LIST=
CHROMIUM_CLEAR_CACHE=false
CHROMIUM_CLEAR_COOKIES=false
CHROMIUM_DISABLE_JAVASCRIPT=false
CHROMIUM_DISABLE_ROUTES=false

# LibreOffice
LIBREOFFICE_RESTART_AFTER=10
LIBREOFFICE_MAX_QUEUE_SIZE=0
LIBREOFFICE_IDLE_SHUTDOWN_TIMEOUT=30s
LIBREOFFICE_AUTO_START=false
LIBREOFFICE_START_TIMEOUT=20s
LIBREOFFICE_ALLOW_LIST=
LIBREOFFICE_DENY_LIST=
LIBREOFFICE_DISABLE_ROUTES=false

# PDF engines
PDFENGINES_MERGE_ENGINES=
PDFENGINES_SPLIT_ENGINES=
PDFENGINES_FLATTEN_ENGINES=
PDFENGINES_CONVERT_ENGINES=
PDFENGINES_READ_METADATA_ENGINES=
PDFENGINES_WRITE_METADATA_ENGINES=
PDFENGINES_READ_BOOKMARKS_ENGINES=
PDFENGINES_WRITE_BOOKMARKS_ENGINES=
PDFENGINES_WATERMARK_ENGINES=
PDFENGINES_STAMP_ENGINES=
PDFENGINES_ENCRYPT_ENGINES=
PDFENGINES_ROTATE_ENGINES=
PDFENGINES_EMBED_ENGINES=
PDFENGINES_EMBED_METADATA_ENGINES=
PDFENGINES_FACTUR_X_ENGINES=
PDFENGINES_DISABLE_ROUTES=false

# Logging
LOG_LEVEL=info
LOG_FIELDS_PREFIX=
LOG_STD_FORMAT=auto
LOG_STD_ENABLE_GCP_FIELDS=false
LOG_STD_LEVEL_CASE=upper

# Prometheus
PROMETHEUS_NAMESPACE=gotenberg
PROMETHEUS_COLLECT_INTERVAL=1s
PROMETHEUS_DISABLE_ROUTE_TELEMETRY=false
PROMETHEUS_DISABLE_COLLECT=false
PROMETHEUS_METRICS_PATH=/metrics

# Webhooks
WEBHOOK_ENABLE_SYNC_MODE=false
WEBHOOK_ALLOW_LIST=
WEBHOOK_DENY_LIST=
WEBHOOK_MAX_RETRY=4
WEBHOOK_RETRY_MIN_WAIT=1s
WEBHOOK_RETRY_MAX_WAIT=30s
WEBHOOK_CLIENT_TIMEOUT=30s
WEBHOOK_DISABLE=false

# Gotenberg flags
GOTENBERG_HIDE_BANNER=false
GOTENBERG_GRACEFUL_SHUTDOWN_DURATION=30s
GOTENBERG_BUILD_DEBUG_DATA=false

# OpenTelemetry (disabled — no otel-collector in this deployment)
OTEL_SERVICE_NAME=gotenberg
OTEL_TRACES_EXPORTER=none
OTEL_METRICS_EXPORTER=none
OTEL_LOGS_EXPORTER=none
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_ENDPOINT=
OTEL_EXPORTER_OTLP_INSECURE=true
EOF

chmod 600 /opt/gotenberg/.env

# ── [7/7] Write compose.yaml and start Gotenberg ────────────────────────────
echo "==> [7/7] Write compose.yaml and start gotenberg"

# All $${VAR} below → Terraform leaves them as literal dollar-brace-VAR;
# Docker Compose resolves them at runtime from the .env file above.
cat > /opt/gotenberg/compose.yaml <<EOF
services:
  gotenberg:
    image: $${DOCKER_REGISTRY}/$${DOCKER_REPOSITORY}:$${GOTENBERG_VERSION}
    restart: unless-stopped
    ports:
      - "$${API_PORT}:$${API_PORT}"
    environment:
      GOTENBERG_API_BASIC_AUTH_USERNAME: $${GOTENBERG_API_BASIC_AUTH_USERNAME}
      GOTENBERG_API_BASIC_AUTH_PASSWORD: $${GOTENBERG_API_BASIC_AUTH_PASSWORD}
      OTEL_SERVICE_NAME:                 $${OTEL_SERVICE_NAME}
      OTEL_TRACES_EXPORTER:              $${OTEL_TRACES_EXPORTER}
      OTEL_METRICS_EXPORTER:             $${OTEL_METRICS_EXPORTER}
      OTEL_LOGS_EXPORTER:                $${OTEL_LOGS_EXPORTER}
      OTEL_EXPORTER_OTLP_PROTOCOL:       $${OTEL_EXPORTER_OTLP_PROTOCOL}
      OTEL_EXPORTER_OTLP_ENDPOINT:       $${OTEL_EXPORTER_OTLP_ENDPOINT}
      OTEL_EXPORTER_OTLP_INSECURE:       $${OTEL_EXPORTER_OTLP_INSECURE}
    command:
      - "gotenberg"
      - "--gotenberg-hide-banner=$${GOTENBERG_HIDE_BANNER}"
      - "--gotenberg-graceful-shutdown-duration=$${GOTENBERG_GRACEFUL_SHUTDOWN_DURATION}"
      - "--gotenberg-build-debug-data=$${GOTENBERG_BUILD_DEBUG_DATA}"
      - "--api-port=$${API_PORT}"
      - "--api-port-from-env=$${API_PORT_FROM_ENV}"
      - "--api-bind-ip=$${API_BIND_IP}"
      - "--api-start-timeout=$${API_START_TIMEOUT}"
      - "--api-timeout=$${API_TIMEOUT}"
      - "--api-body-limit=$${API_BODY_LIMIT}"
      - "--api-root-path=$${API_ROOT_PATH}"
      - "--api-correlation-id-header=$${API_CORRELATION_ID_HEADER}"
      - "--api-enable-basic-auth=$${API_ENABLE_BASIC_AUTH}"
      - "--api-download-from-allow-list=$${API_DOWNLOAD_FROM_ALLOW_LIST}"
      - "--api-download-from-deny-list=$${API_DOWNLOAD_FROM_DENY_LIST}"
      - "--api-download-from-max-retry=$${API_DOWNLOAD_FROM_MAX_RETRY}"
      - "--api-disable-download-from=$${API_DISABLE_DOWNLOAD_FROM}"
      - "--api-disable-health-check-route-telemetry=$${API_DISABLE_HEALTH_CHECK_ROUTE_TELEMETRY}"
      - "--api-disable-root-route-telemetry=$${API_DISABLE_ROOT_ROUTE_TELEMETRY}"
      - "--api-disable-debug-route-telemetry=$${API_DISABLE_DEBUG_ROUTE_TELEMETRY}"
      - "--api-disable-version-route-telemetry=$${API_DISABLE_VERSION_ROUTE_TELEMETRY}"
      - "--api-enable-debug-route=$${API_ENABLE_DEBUG_ROUTE}"
      - "--chromium-restart-after=$${CHROMIUM_RESTART_AFTER}"
      - "--chromium-auto-start=$${CHROMIUM_AUTO_START}"
      - "--chromium-max-queue-size=$${CHROMIUM_MAX_QUEUE_SIZE}"
      - "--chromium-idle-shutdown-timeout=$${CHROMIUM_IDLE_SHUTDOWN_TIMEOUT}"
      - "--chromium-max-concurrency=$${CHROMIUM_MAX_CONCURRENCY}"
      - "--chromium-start-timeout=$${CHROMIUM_START_TIMEOUT}"
      - "--chromium-allow-insecure-localhost=$${CHROMIUM_ALLOW_INSECURE_LOCALHOST}"
      - "--chromium-ignore-certificate-errors=$${CHROMIUM_IGNORE_CERTIFICATE_ERRORS}"
      - "--chromium-disable-web-security=$${CHROMIUM_DISABLE_WEB_SECURITY}"
      - "--chromium-allow-file-access-from-files=$${CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES}"
      - "--chromium-host-resolver-rules=$${CHROMIUM_HOST_RESOLVER_RULES}"
      - "--chromium-proxy-server=$${CHROMIUM_PROXY_SERVER}"
      - "--chromium-allow-list=$${CHROMIUM_ALLOW_LIST}"
      - "--chromium-deny-list=$${CHROMIUM_DENY_LIST}"
      - "--chromium-clear-cache=$${CHROMIUM_CLEAR_CACHE}"
      - "--chromium-clear-cookies=$${CHROMIUM_CLEAR_COOKIES}"
      - "--chromium-disable-javascript=$${CHROMIUM_DISABLE_JAVASCRIPT}"
      - "--chromium-disable-routes=$${CHROMIUM_DISABLE_ROUTES}"
      - "--libreoffice-restart-after=$${LIBREOFFICE_RESTART_AFTER}"
      - "--libreoffice-max-queue-size=$${LIBREOFFICE_MAX_QUEUE_SIZE}"
      - "--libreoffice-idle-shutdown-timeout=$${LIBREOFFICE_IDLE_SHUTDOWN_TIMEOUT}"
      - "--libreoffice-auto-start=$${LIBREOFFICE_AUTO_START}"
      - "--libreoffice-start-timeout=$${LIBREOFFICE_START_TIMEOUT}"
      - "--libreoffice-allow-list=$${LIBREOFFICE_ALLOW_LIST}"
      - "--libreoffice-deny-list=$${LIBREOFFICE_DENY_LIST}"
      - "--libreoffice-disable-routes=$${LIBREOFFICE_DISABLE_ROUTES}"
      - "--log-level=$${LOG_LEVEL}"
      - "--log-fields-prefix=$${LOG_FIELDS_PREFIX}"
      - "--log-std-format=$${LOG_STD_FORMAT}"
      - "--log-std-enable-gcp-fields=$${LOG_STD_ENABLE_GCP_FIELDS}"
      - "--log-std-level-case=$${LOG_STD_LEVEL_CASE}"
      - "--pdfengines-merge-engines=$${PDFENGINES_MERGE_ENGINES}"
      - "--pdfengines-split-engines=$${PDFENGINES_SPLIT_ENGINES}"
      - "--pdfengines-flatten-engines=$${PDFENGINES_FLATTEN_ENGINES}"
      - "--pdfengines-convert-engines=$${PDFENGINES_CONVERT_ENGINES}"
      - "--pdfengines-read-metadata-engines=$${PDFENGINES_READ_METADATA_ENGINES}"
      - "--pdfengines-write-metadata-engines=$${PDFENGINES_WRITE_METADATA_ENGINES}"
      - "--pdfengines-read-bookmarks-engines=$${PDFENGINES_READ_BOOKMARKS_ENGINES}"
      - "--pdfengines-write-bookmarks-engines=$${PDFENGINES_WRITE_BOOKMARKS_ENGINES}"
      - "--pdfengines-watermark-engines=$${PDFENGINES_WATERMARK_ENGINES}"
      - "--pdfengines-stamp-engines=$${PDFENGINES_STAMP_ENGINES}"
      - "--pdfengines-encrypt-engines=$${PDFENGINES_ENCRYPT_ENGINES}"
      - "--pdfengines-rotate-engines=$${PDFENGINES_ROTATE_ENGINES}"
      - "--pdfengines-embed-engines=$${PDFENGINES_EMBED_ENGINES}"
      - "--pdfengines-embed-metadata-engines=$${PDFENGINES_EMBED_METADATA_ENGINES}"
      - "--pdfengines-factur-x-engines=$${PDFENGINES_FACTUR_X_ENGINES}"
      - "--pdfengines-disable-routes=$${PDFENGINES_DISABLE_ROUTES}"
      - "--prometheus-namespace=$${PROMETHEUS_NAMESPACE}"
      - "--prometheus-collect-interval=$${PROMETHEUS_COLLECT_INTERVAL}"
      - "--prometheus-disable-route-telemetry=$${PROMETHEUS_DISABLE_ROUTE_TELEMETRY}"
      - "--prometheus-disable-collect=$${PROMETHEUS_DISABLE_COLLECT}"
      - "--prometheus-metrics-path=$${PROMETHEUS_METRICS_PATH}"
      - "--webhook-enable-sync-mode=$${WEBHOOK_ENABLE_SYNC_MODE}"
      - "--webhook-allow-list=$${WEBHOOK_ALLOW_LIST}"
      - "--webhook-deny-list=$${WEBHOOK_DENY_LIST}"
      - "--webhook-max-retry=$${WEBHOOK_MAX_RETRY}"
      - "--webhook-retry-min-wait=$${WEBHOOK_RETRY_MIN_WAIT}"
      - "--webhook-retry-max-wait=$${WEBHOOK_RETRY_MAX_WAIT}"
      - "--webhook-client-timeout=$${WEBHOOK_CLIENT_TIMEOUT}"
      - "--webhook-disable=$${WEBHOOK_DISABLE}"

networks:
  default:
    enable_ipv6: false
EOF

docker compose -f /opt/gotenberg/compose.yaml --env-file /opt/gotenberg/.env up -d gotenberg

echo "==> Bootstrap complete. Gotenberg starting on port 3000."
