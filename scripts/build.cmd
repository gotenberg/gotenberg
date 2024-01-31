set DOCKER_REPO_GH=ghcr.io/onebrief
set GOTENBERG_VERSION=8.0.1

set GOLANG_VERSION=1.21
set DOCKER_REPOSITORY=onebrief
set GOTENBERG_USER_GID=1001
set GOTENBERG_USER_UID=1001
:: See https://github.com/googlefonts/noto-emoji/releases.
set NOTO_COLOR_EMOJI_VERSION=v2.034 
:: See https://gitlab.com/pdftk-java/pdftk/-/releases - Binary package.
set PDFTK_VERSION=v3.3.2 
:: See https://github.com/golangci/golangci-lint/releases.
set GOLANGCI_LINT_VERSION=v1.45.0 


@REM -t %DOCKER_REPO_GH%/gotenberg:latest ^
@REM -t %DOCKER_REPO_GH%/gotenberg:%GOTENBERG_VERSION% ^
@REM -t gotenberg:cookies ^

docker build ^
  --build-arg GOLANG_VERSION=%GOLANG_VERSION% ^
  --build-arg GOTENBERG_VERSION=%GOTENBERG_VERSION% ^
  --build-arg GOTENBERG_USER_GID=%GOTENBERG_USER_GID% ^
  --build-arg GOTENBERG_USER_UID=%GOTENBERG_USER_UID% ^
  --build-arg NOTO_COLOR_EMOJI_VERSION=%NOTO_COLOR_EMOJI_VERSION% ^
  --build-arg PDFTK_VERSION=%PDFTK_VERSION% ^
  --platform linux/amd64 ^
  -t %DOCKER_REPO_GH%/gotenberg:%GOTENBERG_VERSION% ^
  -f build/Dockerfile .

