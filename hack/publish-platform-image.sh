#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

usage() {
	cat <<'EOF'
Usage:
  hack/publish-platform-image.sh

Required environment variables:
  IMAGE            Fully-qualified image reference to publish.
  CONTROLLER_MAIN  Go entrypoint passed to the Docker build.

Optional environment variables:
  PLATFORMS        Comma-separated target platforms. Accepts linux_amd64 or linux/amd64 syntax.
  DOCKER_BIN       Docker CLI to use. Defaults to docker.
  USE_DOCKER_PLATFORM  When true, pass --platform to docker build. Defaults to false.
  CGO_ENABLED      CGO setting passed to the Docker build. Defaults to 0.
  GOEXPERIMENT     GOEXPERIMENT setting passed to the Docker build. Defaults to empty.
  SKIP_FIPS        OSOK_SKIP_FIPS value baked into the image. Defaults to true.
EOF
}

image=${IMAGE:-}
controller_main=${CONTROLLER_MAIN:-}
platforms=${PLATFORMS:-linux_amd64}
docker_bin=${DOCKER_BIN:-docker}
use_docker_platform=${USE_DOCKER_PLATFORM:-false}
cgo_enabled=${CGO_ENABLED:-0}
goexperiment=${GOEXPERIMENT:-}
skip_fips=${SKIP_FIPS:-true}

if [[ -z "${image}" || -z "${controller_main}" ]]; then
	usage >&2
	exit 1
fi

normalize_platforms() {
	local raw_platforms=$1
	local normalized=()
	local raw trimmed

	IFS=',' read -r -a raw_items <<< "${raw_platforms}"
	for raw in "${raw_items[@]}"; do
		trimmed=$(echo "${raw}" | xargs)
		[[ -n "${trimmed}" ]] || continue
		trimmed=${trimmed//_//}
		if [[ "${trimmed}" != */* ]]; then
			echo "invalid platform '${raw}'; use linux_amd64 or linux/amd64" >&2
			exit 1
		fi
		normalized+=("${trimmed}")
	done

	if [[ ${#normalized[@]} -eq 0 ]]; then
		echo "PLATFORMS must contain at least one platform" >&2
		exit 1
	fi

	printf '%s\n' "${normalized[@]}"
}

normalized_platforms=()
while IFS= read -r platform; do
	normalized_platforms+=("${platform}")
done < <(normalize_platforms "${platforms}")

arch_images=()
for platform in "${normalized_platforms[@]}"; do
	os=${platform%%/*}
	arch=${platform##*/}
	platform_image="${image}-${arch}"
	echo ">>> Building ${platform_image} for ${platform}"
	if [[ "${use_docker_platform}" == "true" ]]; then
		"${docker_bin}" build \
			--platform "${platform}" \
			--build-arg CONTROLLER_MAIN="${controller_main}" \
			--build-arg TARGETOS="${os}" \
			--build-arg TARGETARCH="${arch}" \
			--build-arg CGO_ENABLED="${cgo_enabled}" \
			--build-arg GOEXPERIMENT="${goexperiment}" \
			--build-arg SKIP_FIPS="${skip_fips}" \
			-t "${platform_image}" \
			"${ROOT_DIR}"
	else
		"${docker_bin}" build \
			--build-arg CONTROLLER_MAIN="${controller_main}" \
			--build-arg TARGETOS="${os}" \
			--build-arg TARGETARCH="${arch}" \
			--build-arg CGO_ENABLED="${cgo_enabled}" \
			--build-arg GOEXPERIMENT="${goexperiment}" \
			--build-arg SKIP_FIPS="${skip_fips}" \
			-t "${platform_image}" \
			"${ROOT_DIR}"
	fi
	echo ">>> Pushing ${platform_image}"
	"${docker_bin}" push "${platform_image}"
	arch_images+=("${platform_image}")
done

"${docker_bin}" manifest rm "${image}" >/dev/null 2>&1 || true
"${docker_bin}" manifest create "${image}" "${arch_images[@]}"

for platform in "${normalized_platforms[@]}"; do
	os=${platform%%/*}
	arch=${platform##*/}
	"${docker_bin}" manifest annotate "${image}" "${image}-${arch}" --os "${os}" --arch "${arch}"
done

echo ">>> Pushing manifest ${image}"
"${docker_bin}" manifest push "${image}"
