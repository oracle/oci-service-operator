#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

usage() {
	cat <<'EOF'
Usage:
  hack/package-olm.sh bundle <group>

Required environment variables:
  VERSION         Bundle version. A leading "v" is allowed and will be stripped.

Optional environment variables:
  CONTROLLER_IMG  Controller image to embed in the package install manifest.
  KUSTOMIZE       Kustomize binary path.
  OPERATOR_SDK    Operator SDK binary path.
EOF
}

if [[ $# -ne 2 ]]; then
	usage
	exit 1
fi

command_name=$1
group=$2

package_dir="${ROOT_DIR}/packages/${group}"
metadata_file="${package_dir}/metadata.env"

if [[ ! -f "${metadata_file}" ]]; then
	echo "unknown package group: ${group}" >&2
	exit 1
fi

version=${VERSION:-}
if [[ -z "${version}" ]]; then
	echo "VERSION must be set" >&2
	exit 1
fi
version=${version#v}

kustomize=${KUSTOMIZE:-kustomize}
operator_sdk=${OPERATOR_SDK:-operator-sdk}

# shellcheck disable=SC1090
set -a
source "${metadata_file}"
set +a

: "${PACKAGE_NAME:?missing PACKAGE_NAME in ${metadata_file}}"

write_manifests_kustomization() {
	local file=$1

	cat >"${file}" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../package-install.yaml
- ../scorecard
EOF
}

sync_generated_bundle() {
	local source_root=$1

	rm -rf "${ROOT_DIR}/bundle"
	mkdir -p "${ROOT_DIR}/bundle"
	cp -R "${source_root}/bundle/." "${ROOT_DIR}/bundle/"
	cp "${source_root}/bundle.Dockerfile" "${ROOT_DIR}/bundle.Dockerfile"
}

rewrite_bundle_identity() {
	local bundle_root=$1
	local old_csv new_csv

	old_csv="${bundle_root}/bundle/manifests/oci-service-operator.clusterserviceversion.yaml"
	new_csv="${bundle_root}/bundle/manifests/${PACKAGE_NAME}.clusterserviceversion.yaml"

	perl -0pi -e "s/operators\\.operatorframework\\.io\\.bundle\\.package\\.v1: oci-service-operator/operators.operatorframework.io.bundle.package.v1: ${PACKAGE_NAME}/g" \
		"${bundle_root}/bundle/metadata/annotations.yaml"
	perl -0pi -e "s/operators\\.operatorframework\\.io\\.bundle\\.package\\.v1=oci-service-operator/operators.operatorframework.io.bundle.package.v1=${PACKAGE_NAME}/g" \
		"${bundle_root}/bundle.Dockerfile"
	perl -0pi -e "s/name: oci-service-operator\\.v/name: ${PACKAGE_NAME}.v/g" \
		"${old_csv}"
	mv "${old_csv}" "${new_csv}"
}

generate_bundle() {
	local workdir manifests_dir bases_dir scorecard_dir

	workdir="${ROOT_DIR}/dist/.work/${group}-olm"
	rm -rf "${workdir}"
	mkdir -p "${workdir}"
	cp "${ROOT_DIR}/PROJECT" "${workdir}/PROJECT"

	CONTROLLER_GEN_RUNNER=${CONTROLLER_GEN_RUNNER:-} \
	CONTROLLER_GEN=${CONTROLLER_GEN:-} \
	KUSTOMIZE="${kustomize}" \
	CONTROLLER_IMG="${CONTROLLER_IMG:-}" \
	OUT="${workdir}/package-install.yaml" \
	"${ROOT_DIR}/hack/package.sh" render "${group}"

	manifests_dir="${workdir}/config/manifests"
	bases_dir="${manifests_dir}/bases"
	scorecard_dir="${workdir}/config/scorecard"

	mkdir -p "${bases_dir}" "${workdir}/config"
	write_manifests_kustomization "${manifests_dir}/kustomization.yaml"
	cp -R "${ROOT_DIR}/config/scorecard" "${scorecard_dir}"
	cp "${ROOT_DIR}/config/manifests/bases/oci-service-operator.clusterserviceversion.yaml" \
		"${bases_dir}/oci-service-operator.clusterserviceversion.yaml"

	(
		cd "${workdir}"
		"${operator_sdk}" generate kustomize manifests -q --interactive=false --package "${PACKAGE_NAME}"
		"${kustomize}" build --load-restrictor LoadRestrictionsNone config/manifests | \
			"${operator_sdk}" generate bundle -q --overwrite --version "${version}"
	)

	rewrite_bundle_identity "${workdir}"
	sync_generated_bundle "${workdir}"
}

case "${command_name}" in
bundle)
	generate_bundle
	;;
*)
	usage
	exit 1
	;;
esac
