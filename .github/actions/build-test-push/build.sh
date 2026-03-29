#!/bin/bash

# Exit early.
# See: https://www.gnu.org/savannah-checkouts/gnu/bash/manual/bash.html#The-Set-Builtin.
set -e

# Source dot env file.
source .env

# Arguments.
version=""
platform=""
alternate_repository=""
dry_run=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --version)
      version="${2//v/}"
      shift 2
      ;;
    --platform)
      platform="$2"
      shift 2
      ;;
    --alternate-repository)
      alternate_repository="$2"
      shift 2
      ;;
    --dry-run)
      dry_run="$2"
      shift 2
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

echo "Build and push 👷"
echo

echo "Gotenberg version: $version"
echo "Target platform: $platform"

if [ -n "$alternate_repository" ]; then
  DOCKER_REPOSITORY=$alternate_repository
  echo "⚠️ Using $alternate_repository for DOCKER_REPOSITORY"
fi

if [ "$dry_run" = "true" ]; then
  echo "🚧 Dry run"
fi

# Build tags arrays.
tags=()
tags_chromium=()
tags_libreoffice=()
tags_cloud_run=()
tags_cloud_run_chromium=()
tags_cloud_run_libreoffice=()
tags_aws_lambda=()
tags_aws_lambda_chromium=()
tags_aws_lambda_libreoffice=()

IFS='/' read -ra arch <<< "$platform"
IFS='.' read -ra semver <<< "$version"

if [ "${#semver[@]}" -eq 3 ]; then
  echo
  echo "Semver version detected"

  major="${semver[0]}"
  minor="${semver[1]}"
  patch="${semver[2]}"

  for suffix in "latest" "$major" "$major.$minor" "$major.$minor.$patch"; do
    tags+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-${arch[1]}")
    tags_chromium+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-chromium-${arch[1]}")
    tags_libreoffice+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-libreoffice-${arch[1]}")
  done

  if [ "$platform" = "linux/amd64" ]; then
    for suffix in "latest" "$major" "$major.$minor" "$major.$minor.$patch"; do
      tags_cloud_run+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-cloudrun")
      tags_cloud_run_chromium+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-chromium-cloudrun")
      tags_cloud_run_libreoffice+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-libreoffice-cloudrun")
    done
  fi

  if [ "$platform" = "linux/amd64" ] || [ "$platform" = "linux/arm64" ]; then
    for suffix in "latest" "$major" "$major.$minor" "$major.$minor.$patch"; do
      tags_aws_lambda+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-aws-lambda-${arch[1]}")
      tags_aws_lambda_chromium+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-chromium-aws-lambda-${arch[1]}")
      tags_aws_lambda_libreoffice+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$suffix-libreoffice-aws-lambda-${arch[1]}")
    done
  fi
else
  echo
  echo "Non-semver version detected, fallback to $version"

  tags+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-${arch[1]}")
  tags_chromium+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-chromium-${arch[1]}")
  tags_libreoffice+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-libreoffice-${arch[1]}")

  if [ "$platform" = "linux/amd64" ]; then
    tags_cloud_run+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-cloudrun")
    tags_cloud_run_chromium+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-chromium-cloudrun")
    tags_cloud_run_libreoffice+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-libreoffice-cloudrun")
  fi

  if [ "$platform" = "linux/amd64" ] || [ "$platform" = "linux/arm64" ]; then
    tags_aws_lambda+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-aws-lambda-${arch[1]}")
    tags_aws_lambda_chromium+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-chromium-aws-lambda-${arch[1]}")
    tags_aws_lambda_libreoffice+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-libreoffice-aws-lambda-${arch[1]}")
  fi
fi

tags_flags=()
tags_chromium_flags=()
tags_libreoffice_flags=()
tags_cloud_run_flags=()
tags_cloud_run_chromium_flags=()
tags_cloud_run_libreoffice_flags=()
tags_aws_lambda_flags=()
tags_aws_lambda_chromium_flags=()
tags_aws_lambda_libreoffice_flags=()

echo "Will use the following tags:"
for tag in "${tags[@]}"; do
  tags_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_chromium[@]}"; do
  tags_chromium_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_libreoffice[@]}"; do
  tags_libreoffice_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_cloud_run[@]}"; do
  tags_cloud_run_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_cloud_run_chromium[@]}"; do
  tags_cloud_run_chromium_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_cloud_run_libreoffice[@]}"; do
  tags_cloud_run_libreoffice_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_aws_lambda[@]}"; do
  tags_aws_lambda_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_aws_lambda_chromium[@]}"; do
  tags_aws_lambda_chromium_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_aws_lambda_libreoffice[@]}"; do
  tags_aws_lambda_libreoffice_flags+=("-t" "$tag")
  echo "- $tag"
done
echo

# Build images.
run_cmd() {
  local cmd="$1"

  if [ "$dry_run" = "true" ]; then
    echo "🚧 Dry run - would run the following command:"
    echo "$cmd"
    echo
  else
    echo "⚙️ Running command:"
    echo "$cmd"
    eval "$cmd"
    echo
  fi
}

join() {
  local delimiter="$1"
  shift
  local IFS="$delimiter"
  echo "$*"
}

no_arch_tag="$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version"

# Full variant.
cmd="docker buildx build \
    --target gotenberg \
    --build-arg GOTENBERG_VERSION=$version \
    --platform $platform \
    --load \
    ${tags_flags[*]} \
    -t $no_arch_tag \
    -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
"
run_cmd "$cmd"

# Chromium only variant.
cmd="docker buildx build \
    --target gotenberg-chromium \
    --build-arg GOTENBERG_VERSION=$version \
    --platform $platform \
    --load \
    ${tags_chromium_flags[*]} \
    -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
"
run_cmd "$cmd"

# LibreOffice only variant.
cmd="docker buildx build \
    --target gotenberg-libreoffice \
    --build-arg GOTENBERG_VERSION=$version \
    --platform $platform \
    --load \
    ${tags_libreoffice_flags[*]} \
    -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
"
run_cmd "$cmd"

# Cloud Run variants (amd64 only).
if [ "$platform" = "linux/amd64" ]; then
  cmd="docker buildx build \
      --target gotenberg-cloudrun \
      --build-arg GOTENBERG_VERSION=$version \
      --platform $platform \
      --load \
      ${tags_cloud_run_flags[*]} \
      -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"

  cmd="docker buildx build \
      --target gotenberg-cloudrun-chromium \
      --build-arg GOTENBERG_VERSION=$version \
      --platform $platform \
      --load \
      ${tags_cloud_run_chromium_flags[*]} \
      -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"

  cmd="docker buildx build \
      --target gotenberg-cloudrun-libreoffice \
      --build-arg GOTENBERG_VERSION=$version \
      --platform $platform \
      --load \
      ${tags_cloud_run_libreoffice_flags[*]} \
      -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"
fi

# AWS Lambda variants (amd64 + arm64 only).
if [ "$platform" = "linux/amd64" ] || [ "$platform" = "linux/arm64" ]; then
  cmd="docker buildx build \
      --target gotenberg-aws-lambda \
      --build-arg GOTENBERG_VERSION=$version \
      --platform $platform \
      --load \
      ${tags_aws_lambda_flags[*]} \
      -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"

  cmd="docker buildx build \
      --target gotenberg-aws-lambda-chromium \
      --build-arg GOTENBERG_VERSION=$version \
      --platform $platform \
      --load \
      ${tags_aws_lambda_chromium_flags[*]} \
      -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"

  cmd="docker buildx build \
      --target gotenberg-aws-lambda-libreoffice \
      --build-arg GOTENBERG_VERSION=$version \
      --platform $platform \
      --load \
      ${tags_aws_lambda_libreoffice_flags[*]} \
      -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"
fi

echo "✅ Done!"
echo "tags=$(join "," "${tags[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_chromium=$(join "," "${tags_chromium[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_libreoffice=$(join "," "${tags_libreoffice[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_cloud_run=$(join "," "${tags_cloud_run[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_cloud_run_chromium=$(join "," "${tags_cloud_run_chromium[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_cloud_run_libreoffice=$(join "," "${tags_cloud_run_libreoffice[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_aws_lambda=$(join "," "${tags_aws_lambda[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_aws_lambda_chromium=$(join "," "${tags_aws_lambda_chromium[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_aws_lambda_libreoffice=$(join "," "${tags_aws_lambda_libreoffice[@]}")" >> "$GITHUB_OUTPUT"
exit 0
