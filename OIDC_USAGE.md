# OIDC Authentication with Auth0 for Gotenberg

This document describes how to use OIDC authentication with Auth0 in Gotenberg, with a focus on Docker deployments.

## Docker Configuration (Recommended)

Gotenberg is typically deployed as a Docker container. The project provides Dockerfiles in the `build/` directory for building custom images.

### Building Gotenberg with OIDC Support

Since OIDC support is now built into Gotenberg, you can use the existing Dockerfile to build an image with OIDC capabilities:

```bash
# Build the main Gotenberg image with OIDC support
docker build -f build/Dockerfile -t gotenberg-oidc:latest .

# For production builds with version tags
docker build -f build/Dockerfile \
  --build-arg GOTENBERG_VERSION=8.0.0 \
  -t gotenberg-oidc:8.0.0 .
```

### Running Pre-built Images with OIDC

```bash
# Run official Gotenberg image with OIDC authentication enabled
docker run --rm -p 3000:3000 \
  -e GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/" \
  -e GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier" \
  gotenberg/gotenberg:8 \
  gotenberg --api-enable-oidc-auth
```

### Running Your Built Image

```bash
# Run your custom built image
docker run --rm -p 3000:3000 \
  -e GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/" \
  -e GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier" \
  gotenberg-oidc:latest \
  gotenberg --api-enable-oidc-auth --log-level=info
```

### Specialized Build Variants

Following the existing pattern of specialized Dockerfiles, you can create an OIDC-specific variant:

**build/Dockerfile.oidc:**
```dockerfile
ARG DOCKER_REGISTRY
ARG DOCKER_REPOSITORY  
ARG GOTENBERG_VERSION

FROM $DOCKER_REGISTRY/$DOCKER_REPOSITORY:$GOTENBERG_VERSION

USER root

# OIDC-specific environment variables
ENV GOTENBERG_API_ENABLE_OIDC_AUTH=true
ENV GOTENBERG_API_OIDC_JWKS_CACHE_DURATION=1h

USER gotenberg

# Default CMD with OIDC enabled
CMD ["gotenberg", "--api-enable-oidc-auth", "--log-level=info"]
```

Build and run:
```bash
# Build OIDC variant
docker build -f build/Dockerfile.oidc \
  --build-arg DOCKER_REGISTRY=gotenberg \
  --build-arg DOCKER_REPOSITORY=gotenberg \
  --build-arg GOTENBERG_VERSION=8 \
  -t gotenberg-oidc:latest .

# Run with just environment variables
docker run --rm -p 3000:3000 \
  -e GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/" \
  -e GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier" \
  gotenberg-oidc:latest
```

### Environment Variables Reference

| Environment Variable | Description | Example |
|---------------------|-------------|---------|
| `GOTENBERG_API_OIDC_ISSUER` | Auth0 tenant URL | `https://your-tenant.auth0.com/` |
| `GOTENBERG_API_OIDC_AUDIENCE` | API identifier from Auth0 | `your-api-identifier` |
| `GOTENBERG_API_OIDC_JWKS_URL` | JWKS endpoint (optional) | `https://your-tenant.auth0.com/.well-known/jwks.json` |

## Binary Configuration (Development)

For local development or binary deployments:

### Command Line Flags

```bash
# Enable OIDC authentication
./gotenberg --api-enable-oidc-auth \
  --api-oidc-issuer="https://your-tenant.auth0.com/" \
  --api-oidc-audience="your-api-identifier" \
  --api-oidc-jwks-url="https://your-tenant.auth0.com/.well-known/jwks.json" \
  --api-oidc-jwks-cache-duration="1h"
```

### Environment Variables

```bash
# Set environment variables (these override command line flags)
export GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/"
export GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier"
export GOTENBERG_API_OIDC_JWKS_URL="https://your-tenant.auth0.com/.well-known/jwks.json"

# Start with OIDC enabled
./gotenberg --api-enable-oidc-auth
```

## Auth0 Setup

1. **Create an Auth0 Application:**
   - Go to Auth0 Dashboard > Applications
   - Click "Create Application"
   - Choose "Regular Web Applications" or "Machine to Machine Applications"
   - Note the Domain and Client ID

2. **Create an API in Auth0:**
   - Go to Auth0 Dashboard > APIs
   - Click "Create API"
   - Set the Identifier (this will be your audience)
   - Choose RS256 as the signing algorithm

3. **Configure Application:**
   - Grant the application access to your API
   - Configure the required scopes if needed

## Usage Examples

### Using Built Image with cURL

```bash
# Build Gotenberg with OIDC support
docker build -f build/Dockerfile -t gotenberg-oidc:latest .

# Start Gotenberg with OIDC authentication
docker run -d --name gotenberg-oidc -p 3000:3000 \
  -e GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/" \
  -e GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier" \
  gotenberg-oidc:latest \
  gotenberg --api-enable-oidc-auth --log-level=info

# Get a token from Auth0 (replace with your values)
TOKEN=$(curl --request POST \
  --url https://your-tenant.auth0.com/oauth/token \
  --header 'content-type: application/json' \
  --data '{
    "client_id":"your-client-id",
    "client_secret":"your-client-secret",
    "audience":"your-api-identifier",
    "grant_type":"client_credentials"
  }' | jq -r '.access_token')

# Use the token to make requests to Gotenberg
curl -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: multipart/form-data" \
  -F "files=@document.html" \
  http://localhost:3000/forms/chromium/convert/html \
  -o output.pdf

# Clean up
docker stop gotenberg-oidc && docker rm gotenberg-oidc
```

### Complete Testing Workflow

Create these files following the existing build structure:

**build/Dockerfile.oidc:**
```dockerfile
ARG DOCKER_REGISTRY=gotenberg
ARG DOCKER_REPOSITORY=gotenberg
ARG GOTENBERG_VERSION=8

FROM $DOCKER_REGISTRY/$DOCKER_REPOSITORY:$GOTENBERG_VERSION

USER root

# OIDC-specific environment variables with defaults
ENV GOTENBERG_API_OIDC_JWKS_CACHE_DURATION=1h
ENV LOG_LEVEL=info

USER gotenberg

# Default CMD with OIDC enabled
CMD ["gotenberg", "--api-enable-oidc-auth"]
```

**scripts/test-oidc.sh:**
```bash
#!/bin/bash
set -e

# Configuration
IMAGE_NAME="gotenberg-oidc:latest"
CONTAINER_NAME="gotenberg-oidc-test"

# Build the OIDC image
echo "Building Gotenberg OIDC image..."
docker build -f build/Dockerfile.oidc -t $IMAGE_NAME .

# Start container with OIDC configuration
echo "Starting Gotenberg with OIDC authentication..."
docker run -d --name $CONTAINER_NAME -p 3000:3000 \
  -e GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/" \
  -e GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier" \
  $IMAGE_NAME

# Wait for container to be ready
echo "Waiting for Gotenberg to be ready..."
timeout 30s bash -c 'while ! curl -s http://localhost:3000/health > /dev/null; do sleep 1; done'

# Get Auth0 token
echo "Getting Auth0 token..."
TOKEN=$(curl --silent --request POST \
  --url https://your-tenant.auth0.com/oauth/token \
  --header 'content-type: application/json' \
  --data '{
    "client_id":"'${AUTH0_CLIENT_ID}'",
    "client_secret":"'${AUTH0_CLIENT_SECRET}'",
    "audience":"your-api-identifier", 
    "grant_type":"client_credentials"
  }' | jq -r '.access_token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
  echo "Failed to get Auth0 token. Check your credentials."
  exit 1
fi

# Test HTML to PDF conversion
echo "Testing HTML to PDF conversion..."
echo "<html><body><h1>Test Document with OIDC Auth</h1><p>Generated: $(date)</p></body></html>" > test.html

curl -H "Authorization: Bearer $TOKEN" \
  -F "files=@test.html" \
  http://localhost:3000/forms/chromium/convert/html \
  -o test-output.pdf

if [ -f "test-output.pdf" ]; then
  echo "✅ Conversion successful: test-output.pdf"
  ls -la test-output.pdf
else
  echo "❌ Conversion failed"
  exit 1
fi

# Cleanup
echo "Cleaning up..."
docker stop $CONTAINER_NAME && docker rm $CONTAINER_NAME
rm -f test.html

echo "✅ OIDC authentication test completed successfully"
```

**Makefile integration:**
```makefile
# Add to existing Makefile
.PHONY: build-oidc test-oidc

build-oidc:
	docker build -f build/Dockerfile.oidc -t gotenberg-oidc:latest .

test-oidc: build-oidc
	@echo "Testing OIDC authentication..."
	@if [ -z "$$AUTH0_CLIENT_ID" ] || [ -z "$$AUTH0_CLIENT_SECRET" ]; then \
		echo "Please set AUTH0_CLIENT_ID and AUTH0_CLIENT_SECRET environment variables"; \
		exit 1; \
	fi
	chmod +x scripts/test-oidc.sh
	./scripts/test-oidc.sh
```

Run the complete test:
```bash
# Set your Auth0 credentials
export AUTH0_CLIENT_ID="your-client-id"
export AUTH0_CLIENT_SECRET="your-client-secret"

# Run the test (builds image and tests OIDC)
make test-oidc

# Or manually
chmod +x scripts/test-oidc.sh
./scripts/test-oidc.sh
```

### Test with a Mock JWT Token

For testing purposes, you can create a mock JWT token at [jwt.io](https://jwt.io) with the following payload:

```json
{
  "iss": "https://your-tenant.auth0.com/",
  "aud": "your-api-identifier",
  "sub": "test-user",
  "iat": 1640995200,
  "exp": 1640995200,
  "azp": "your-client-id",
  "scope": "read:documents"
}
```

**Note:** For testing only. In production, always use proper tokens from Auth0.

## Production Deployments

### Building for Production

Create production-ready images following the existing build patterns:

**build/Dockerfile.production:**
```dockerfile
ARG GOLANG_VERSION=1.25.0
ARG GOTENBERG_VERSION=8.0.0

# Use multi-stage build from existing Dockerfile
FROM golang:$GOLANG_VERSION AS builder

# Copy the enhanced source with OIDC support
WORKDIR /app
COPY . .

# Build with production optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-s -w -X 'github.com/gotenberg/gotenberg/v8/cmd.Version=$GOTENBERG_VERSION'" \
    -o gotenberg cmd/gotenberg/main.go

# Production image based on existing Dockerfile structure
FROM debian:13-slim

# Copy from existing Dockerfile setup (fonts, dependencies, etc.)
# ... (use existing Dockerfile as base)

# Copy our enhanced binary
COPY --from=builder /app/gotenberg /usr/bin/

# Production-specific OIDC defaults
ENV GOTENBERG_API_OIDC_JWKS_CACHE_DURATION=1h
ENV LOG_LEVEL=info

USER gotenberg
WORKDIR /home/gotenberg

EXPOSE 3000
ENTRYPOINT [ "/usr/bin/tini", "--" ]
CMD ["gotenberg", "--api-enable-oidc-auth"]
```

Build production image:
```bash
# Build production image with OIDC support
docker build -f build/Dockerfile.production \
  --build-arg GOTENBERG_VERSION=8.0.0 \
  -t gotenberg-oidc:8.0.0 .

# Tag for deployment
docker tag gotenberg-oidc:8.0.0 your-registry.com/gotenberg-oidc:8.0.0
docker push your-registry.com/gotenberg-oidc:8.0.0
```

### Kubernetes Deployment

Deploy your built image to Kubernetes:

**k8s/gotenberg-oidc.yaml:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gotenberg-oidc
  labels:
    app: gotenberg-oidc
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gotenberg-oidc
  template:
    metadata:
      labels:
        app: gotenberg-oidc
    spec:
      containers:
      - name: gotenberg
        image: your-registry.com/gotenberg-oidc:8.0.0
        ports:
        - containerPort: 3000
        env:
        - name: GOTENBERG_API_OIDC_ISSUER
          valueFrom:
            secretKeyRef:
              name: gotenberg-oidc-config
              key: issuer
        - name: GOTENBERG_API_OIDC_AUDIENCE
          valueFrom:
            secretKeyRef:
              name: gotenberg-oidc-config
              key: audience
        resources:
          limits:
            memory: "1Gi"
            cpu: "500m"
          requests:
            memory: "512Mi"
            cpu: "250m"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Secret
metadata:
  name: gotenberg-oidc-config
type: Opaque
stringData:
  issuer: "https://your-tenant.auth0.com/"
  audience: "your-api-identifier"
---
apiVersion: v1
kind: Service
metadata:
  name: gotenberg-service
spec:
  selector:
    app: gotenberg-oidc
  ports:
    - protocol: TCP
      port: 80
      targetPort: 3000
  type: LoadBalancer
```

Deploy using the existing CI/CD pattern:
```bash
# Build and deploy script (similar to existing patterns)
#!/bin/bash
set -e

VERSION=${1:-"latest"}
REGISTRY=${DOCKER_REGISTRY:-"your-registry.com"}

# Build production image
echo "Building Gotenberg OIDC production image..."
docker build -f build/Dockerfile.production \
  --build-arg GOTENBERG_VERSION=$VERSION \
  -t gotenberg-oidc:$VERSION .

# Tag for registry
docker tag gotenberg-oidc:$VERSION $REGISTRY/gotenberg-oidc:$VERSION

# Push to registry
docker push $REGISTRY/gotenberg-oidc:$VERSION

# Deploy to Kubernetes
kubectl apply -f k8s/gotenberg-oidc.yaml

echo "Deployment completed successfully"
```

### Cloud Run Deployment

Create a Cloud Run specific Dockerfile following the existing `Dockerfile.cloudrun` pattern:

**build/Dockerfile.cloudrun-oidc:**
```dockerfile
ARG DOCKER_REGISTRY
ARG DOCKER_REPOSITORY
ARG GOTENBERG_VERSION

FROM $DOCKER_REGISTRY/$DOCKER_REPOSITORY:$GOTENBERG_VERSION

USER root

# Cloud Run specific settings from existing Dockerfile.cloudrun
RUN chown gotenberg: /usr/bin/tini

# OIDC and Cloud Run specific environment variables
ENV API_PORT_FROM_ENV=PORT
ENV GOTENBERG_API_OIDC_JWKS_CACHE_DURATION=1h
ENV LOG_ENABLE_GCP_FIELDS=true
ENV WEBHOOK_ENABLE_SYNC_MODE=true
ENV GOTENBERG_BUILD_DEBUG_DATA=false

USER gotenberg

# Enable OIDC by default for Cloud Run
CMD ["gotenberg", "--api-enable-oidc-auth"]
```

Deploy to Cloud Run:
```bash
# Build Cloud Run image
docker build -f build/Dockerfile.cloudrun-oidc \
  --build-arg DOCKER_REGISTRY=gotenberg \
  --build-arg DOCKER_REPOSITORY=gotenberg \
  --build-arg GOTENBERG_VERSION=8 \
  -t gotenberg-oidc-cloudrun:latest .

# Push to Google Container Registry
docker tag gotenberg-oidc-cloudrun:latest gcr.io/your-project/gotenberg-oidc:latest
docker push gcr.io/your-project/gotenberg-oidc:latest

# Deploy to Cloud Run
gcloud run deploy gotenberg-oidc \
  --image gcr.io/your-project/gotenberg-oidc:latest \
  --platform managed \
  --region us-central1 \
  --set-env-vars GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/" \
  --set-env-vars GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier" \
  --allow-unauthenticated
```

## Features

- **Automatic JWKS Discovery:** If `--api-oidc-jwks-url` is not provided, the JWKS URL is automatically discovered from the issuer
- **Token Caching:** JWKS keys are cached for the duration specified by `--api-oidc-jwks-cache-duration`
- **Full Validation:** Validates issuer, audience, expiration, and token signature
- **Mutual Exclusion:** Cannot enable both basic auth and OIDC auth simultaneously
- **Context Storage:** Validated JWT claims are stored in the request context for use by handlers

## Error Responses

| Status Code | Error | Description |
|-------------|--------|-------------|
| 401 | Authorization header is required | No Authorization header provided |
| 401 | Authorization header must start with 'Bearer ' | Invalid header format |
| 401 | Bearer token is required | Empty token |
| 401 | Invalid token | JWT parsing/validation failed |
| 401 | Invalid token issuer | Token issuer doesn't match configured issuer |
| 401 | Invalid token audience | Token audience doesn't match configured audience |
| 401 | Token has expired | Token is past its expiration time |
| 401 | Token used before issued | Token is being used before its issued at time |
| 401 | Invalid token claims | Token has invalid or missing claims |

## Troubleshooting

### Common Issues

1. **"Invalid token" errors:**
   - Check that your token is valid at [jwt.io](https://jwt.io)
   - Ensure the token is not expired
   - Verify the issuer and audience match your configuration

2. **"Key with ID 'xxx' not found" errors:**
   - Verify the JWKS URL is correct
   - Check that the token's `kid` (key ID) exists in the JWKS
   - Ensure network connectivity to the JWKS endpoint

3. **"Unexpected signing method" errors:**
   - Gotenberg expects RS256 signed tokens
   - Check your Auth0 API settings to ensure RS256 is used

### Debug Mode

#### Docker Container Debugging

Enable debug logging for Docker containers using the built images:

```bash
# Run built image with debug logging
docker run --rm -p 3000:3000 \
  -e GOTENBERG_API_OIDC_ISSUER="https://your-tenant.auth0.com/" \
  -e GOTENBERG_API_OIDC_AUDIENCE="your-api-identifier" \
  gotenberg-oidc:latest \
  gotenberg --api-enable-oidc-auth --log-level=debug

# Using debug variant Dockerfile
# build/Dockerfile.debug:
# FROM gotenberg-oidc:latest
# CMD ["gotenberg", "--api-enable-oidc-auth", "--log-level=debug"]
```

#### Container Logs and Debugging

```bash
# View real-time logs
docker logs -f gotenberg

# For docker-compose
docker-compose logs -f gotenberg

# Check container status
docker ps

# Exec into running container for debugging
docker exec -it gotenberg sh

# Test connectivity to Auth0 from within container
docker exec gotenberg curl -v https://your-tenant.auth0.com/.well-known/jwks.json
```

#### Kubernetes Debugging

```bash
# View pod logs
kubectl logs -f deployment/gotenberg-oidc

# Describe pod for events and status
kubectl describe pod -l app=gotenberg

# Port forward for local testing
kubectl port-forward service/gotenberg-service 3000:80

# Exec into pod for debugging
kubectl exec -it deployment/gotenberg-oidc -- sh
```

#### Common Docker Issues

1. **Container not starting:**
   ```bash
   # Check container logs for startup errors
   docker logs gotenberg
   ```

2. **Network connectivity issues:**
   ```bash
   # Test network connectivity from container
   docker exec gotenberg ping your-tenant.auth0.com
   docker exec gotenberg nslookup your-tenant.auth0.com
   ```

3. **Environment variables not set:**
   ```bash
   # Check environment variables in running container
   docker exec gotenberg env | grep GOTENBERG
   ```

#### Binary Debug Mode (Development)

Enable debug logging for binary deployments:

```bash
./gotenberg --api-enable-oidc-auth --log-level=debug
```

This will log token validation failures with detailed error messages.

## Security Considerations for Docker Deployments

### Environment Variables and Secrets

1. **Build-time vs Runtime Secrets:**
   ```dockerfile
   # ❌ Bad - secrets in build-time 
   ARG OIDC_SECRET
   ENV GOTENBERG_API_OIDC_ISSUER=$OIDC_SECRET
   
   # ✅ Good - runtime environment variables
   # Pass secrets at runtime via docker run -e or Kubernetes secrets
   ```

2. **Use external secret management:**
   ```bash
   # Using environment files with restricted permissions
   touch .env
   chmod 600 .env
   echo "GOTENBERG_API_OIDC_ISSUER=https://your-tenant.auth0.com/" >> .env
   echo "GOTENBERG_API_OIDC_AUDIENCE=your-api-identifier" >> .env
   
   # Run container with env file
   docker run --env-file .env gotenberg-oidc:latest
   ```

3. **Kubernetes Secrets integration:**
   ```bash
   # Create secret from command line
   kubectl create secret generic gotenberg-oidc-config \
     --from-literal=issuer="https://your-tenant.auth0.com/" \
     --from-literal=audience="your-api-identifier"
   ```

### Container Security

1. **Use the existing non-root user pattern:**
   ```dockerfile
   # Following existing Dockerfile pattern
   ARG GOTENBERG_USER_GID=1001
   ARG GOTENBERG_USER_UID=1001
   
   RUN groupadd --gid "$GOTENBERG_USER_GID" gotenberg && \
       useradd --uid "$GOTENBERG_USER_UID" --gid gotenberg gotenberg
   
   USER gotenberg
   ```

2. **Multi-stage builds for security:**
   ```dockerfile
   # Build stage - contains build tools
   FROM golang:1.25.0 AS builder
   WORKDIR /app
   COPY . .
   RUN go build -o gotenberg cmd/gotenberg/main.go
   
   # Production stage - minimal attack surface
   FROM debian:13-slim
   COPY --from=builder /app/gotenberg /usr/bin/
   USER gotenberg
   ```

3. **Resource limits in deployment:**
   ```bash
   # Docker run with limits
   docker run --memory="1g" --cpus="0.5" \
     -e GOTENBERG_API_OIDC_ISSUER="..." \
     gotenberg-oidc:latest
   ```

### Network Security

1. **Container network isolation:**
   ```bash
   # Create isolated network
   docker network create --driver bridge gotenberg-net
   
   # Run container in isolated network
   docker run --network gotenberg-net \
     -e GOTENBERG_API_OIDC_ISSUER="..." \
     gotenberg-oidc:latest
   ```

2. **Production reverse proxy setup:**
   ```dockerfile
   # build/Dockerfile.nginx-proxy
   FROM nginx:alpine
   COPY nginx.conf /etc/nginx/nginx.conf
   COPY ssl/ /etc/ssl/
   
   # nginx.conf should proxy to gotenberg container
   ```

### Auth0 Configuration Security

1. **Use Machine-to-Machine Applications** for API access
2. **Configure proper scopes** and restrict access  
3. **Use short-lived tokens** when possible
4. **Implement proper CORS policies** in Auth0
5. **Monitor Auth0 logs** for suspicious activity
6. **Use separate Auth0 tenants** for different environments

### Production Security Checklist

```bash
# Security validation script
#!/bin/bash
echo "🔒 Gotenberg OIDC Security Validation"

# Check if running as root
if [ "$(docker exec gotenberg-container whoami)" = "root" ]; then
  echo "❌ Container running as root - security risk"
else
  echo "✅ Container running as non-root user"
fi

# Check resource limits
docker inspect gotenberg-container --format='{{.HostConfig.Memory}}' | grep -q "0" && \
  echo "❌ No memory limit set" || echo "✅ Memory limit configured"

# Check for secrets in environment
docker exec gotenberg-container env | grep -q "SECRET" && \
  echo "❌ Potential secrets in environment variables" || \
  echo "✅ No obvious secrets in environment"

# Validate HTTPS endpoints
curl -s https://your-tenant.auth0.com/.well-known/jwks.json > /dev/null && \
  echo "✅ Auth0 JWKS endpoint accessible via HTTPS" || \
  echo "❌ Cannot reach Auth0 JWKS endpoint"

echo "🔒 Security validation complete"
```

### Monitoring and Alerting

1. **Built-in Prometheus metrics:**
   ```bash
   # Gotenberg exposes metrics by default
   curl http://localhost:3000/metrics
   ```

2. **Container monitoring:**
   ```bash
   # Monitor container health
   docker run -d --name gotenberg-oidc \
     --health-cmd="curl -f http://localhost:3000/health || exit 1" \
     --health-interval=30s \
     --health-timeout=10s \
     --health-retries=3 \
     gotenberg-oidc:latest
   ```
