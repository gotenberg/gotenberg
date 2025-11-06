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
tags_cloud_run=()
tags_aws_lambda=()

IFS='/' read -ra arch <<< "$platform"
IFS='.' read -ra semver <<< "$version"

if [ "${#semver[@]}" -eq 3 ]; then
  echo
  echo "Semver version detected"

  major="${semver[0]}"
  minor="${semver[1]}"
  patch="${semver[2]}"

  tags+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:latest-${arch[1]}")
  tags+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major-${arch[1]}")
  tags+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major.$minor-${arch[1]}")
  tags+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major.$minor.$patch-${arch[1]}")

  if [ "$platform" = "linux/amd64" ]; then
    tags_cloud_run+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:latest-cloudrun")
    tags_cloud_run+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major-cloudrun")
    tags_cloud_run+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major.$minor-cloudrun")
    tags_cloud_run+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major.$minor.$patch-cloudrun")
  fi

  if [ "$platform" = "linux/amd64" ] || [ "$platform" = "linux/arm64" ]; then
    tags_aws_lambda+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:latest-aws-lambda-${arch[1]}")
    tags_aws_lambda+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major-aws-lambda-${arch[1]}")
    tags_aws_lambda+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major.$minor-aws-lambda-${arch[1]}")
    tags_aws_lambda+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$major.$minor.$patch-aws-lambda-${arch[1]}")
  fi
else
  echo
  echo "Non-semver version detected, fallback to $version"

  tags+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-${arch[1]}")
  if [ "$platform" = "linux/amd64" ]; then
    tags_cloud_run+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-cloudrun")
  fi

  if [ "$platform" = "linux/amd64" ] || [ "$platform" = "linux/arm64" ]; then
    tags_aws_lambda+=("$DOCKER_REGISTRY/$DOCKER_REPOSITORY:$version-aws-lambda-${arch[1]}")
  fi
fi

tags_flags=()
tags_cloud_run_flags=()
tags_aws_lambda_flags=()

echo "Will use the following tags:"
for tag in "${tags[@]}"; do
  tags_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_cloud_run[@]}"; do
  tags_cloud_run_flags+=("-t" "$tag")
  echo "- $tag"
done
for tag in "${tags_aws_lambda[@]}"; do
  tags_aws_lambda_flags+=("-t" "$tag")
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

cmd="docker buildx build \
    --build-arg GOTENBERG_VERSION=$version \
    --platform $platform \
    --load \
    ${tags_flags[*]} \
    -t $no_arch_tag \
    -f $DOCKERFILE $DOCKER_BUILD_CONTEXT
"
run_cmd "$cmd"

if [ "$platform" = "linux/amd64" ]; then
  cmd="docker build \
      --build-arg DOCKER_REGISTRY=$DOCKER_REGISTRY \
      --build-arg DOCKER_REPOSITORY=$DOCKER_REPOSITORY \
      --build-arg GOTENBERG_VERSION=$version \
      ${tags_cloud_run_flags[*]} \
      -f $DOCKERFILE_CLOUDRUN $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"
fi

if [ "$platform" = "linux/amd64" ] || [ "$platform" = "linux/arm64" ]; then
  cmd="docker build \
        --build-arg DOCKER_REGISTRY=$DOCKER_REGISTRY \
        --build-arg DOCKER_REPOSITORY=$DOCKER_REPOSITORY \
        --build-arg GOTENBERG_VERSION=$version \
        ${tags_aws_lambda_flags[*]} \
        -f $DOCKERFILE_AWS_LAMBDA $DOCKER_BUILD_CONTEXT
  "
  run_cmd "$cmd"
fi

echo "✅ Done!"
echo "tags=$(join "," "${tags[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_cloud_run=$(join "," "${tags_cloud_run[@]}")" >> "$GITHUB_OUTPUT"
echo "tags_aws_lambda=$(join "," "${tags_aws_lambda[@]}")" >> "$GITHUB_OUTPUT"
exit 0
