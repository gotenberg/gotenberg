# Gotenberg (fork Q10) — build & run

## 1. Construir la imagen

```bash
make build
# equivalente:
docker build --target gotenberg -t gotenberg/gotenberg:snapshot -f build/Dockerfile .
```

## 2. (Opcional) Publicar en un registry

```bash
docker tag gotenberg/gotenberg:snapshot TU_REGISTRY/gotenberg:1.0
docker push TU_REGISTRY/gotenberg:1.0
# en el server: ajustar DOCKER_REGISTRY / DOCKER_REPOSITORY / GOTENBERG_VERSION en .env
```

## 3. Levantar

```bash
# la clave del basic auth se pasa por entorno:
export GOTENBERG_API_BASIC_AUTH_PASSWORD='clave-fuerte'
docker compose up -d gotenberg
```

- Usar `docker compose up`, **no** `make run` (make pisa el `.env` con los defaults de upstream).
- El shell tiene precedencia sobre `.env`, por eso la clave del `export` se aplica.
- Expone el puerto **3000** y arranca con `restart: unless-stopped`.

## 4. Verificar

```bash
curl -u q10-firma:'una-clave-fuerte' http://localhost:3000/health
```
