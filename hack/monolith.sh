#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

usage() {
	cat <<'EOF'
Usage:
  hack/monolith.sh render
EOF
}

if [[ $# -ne 1 ]]; then
	usage
	exit 1
fi

command_name=$1

kustomize=${KUSTOMIZE:-kustomize}
controller_image=${CONTROLLER_IMG:-iad.ocir.io/oracle/oci-service-operator:latest}
output_path=${OUT:-"${ROOT_DIR}/dist/monolith/install.yaml"}

render_workdir() {
	local workdir

	workdir="${ROOT_DIR}/dist/.work/monolith"
	rm -rf "${workdir}"
	mkdir -p "${workdir}"

	cat >"${workdir}/kustomization.yaml" <<'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../../config/default
EOF

	(
		cd "${workdir}"
		"${kustomize}" edit set image "controller=${controller_image}" >/dev/null
	)

	printf '%s\n' "${workdir}"
}

render_install() {
	local workdir

	workdir=$(render_workdir)
	trap 'rm -rf "${workdir}"' RETURN

	if [[ "${output_path}" == "-" ]]; then
		"${kustomize}" build --load-restrictor LoadRestrictionsNone "${workdir}"
	else
		mkdir -p "$(dirname "${output_path}")"
		"${kustomize}" build --load-restrictor LoadRestrictionsNone "${workdir}" >"${output_path}"
	fi
}

case "${command_name}" in
render)
	render_install
	;;
*)
	usage
	exit 1
	;;
esac
