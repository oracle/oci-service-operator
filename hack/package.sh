#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

usage() {
	cat <<'EOF'
Usage:
  hack/package.sh generate <group>
  hack/package.sh render <group>
EOF
}

if [[ $# -lt 2 ]]; then
	usage
	exit 1
fi

command_name=$1
group=$2

package_dir="${ROOT_DIR}/packages/${group}"
metadata_file="${package_dir}/metadata.env"
install_dir="${package_dir}/install"

if [[ ! -f "${metadata_file}" ]]; then
	echo "unknown package group: ${group}" >&2
	exit 1
fi

# shellcheck disable=SC1090
set -a
source "${metadata_file}"
set +a

: "${CRD_PATHS:?missing CRD_PATHS in ${metadata_file}}"
: "${RBAC_PATHS:?missing RBAC_PATHS in ${metadata_file}}"
: "${DEFAULT_CONTROLLER_IMAGE:?missing DEFAULT_CONTROLLER_IMAGE in ${metadata_file}}"

controller_gen=${CONTROLLER_GEN:-controller-gen}
kustomize=${KUSTOMIZE:-kustomize}
controller_image=${CONTROLLER_IMG:-${DEFAULT_CONTROLLER_IMAGE}}
crd_options=${CRD_OPTIONS:-crd:generateEmbeddedObjectMeta=true,allowDangerousTypes=true}

generated_dir="${install_dir}/generated"
generated_crd_dir="${generated_dir}/crd"
generated_crd_bases_dir="${generated_crd_dir}/bases"
generated_rbac_dir="${generated_dir}/rbac"

write_crd_kustomization() {
	local kustomization_file=$1
	local found=0

	{
		echo "apiVersion: kustomize.config.k8s.io/v1beta1"
		echo "kind: Kustomization"
		echo "resources:"
		shopt -s nullglob
		for file in "${generated_crd_bases_dir}"/*.yaml; do
			found=1
			echo "- bases/$(basename "${file}")"
		done
		shopt -u nullglob
	} >"${kustomization_file}"

	if [[ ${found} -eq 0 ]]; then
		echo "no CRDs were generated for group ${group}" >&2
		exit 1
	fi
}

generate_assets() {
	rm -rf "${generated_dir}"
	mkdir -p "${generated_crd_bases_dir}" "${generated_rbac_dir}"

	"${controller_gen}" \
		"${crd_options}" \
		paths="${CRD_PATHS}" \
		output:crd:artifacts:config="${generated_crd_bases_dir}"

	write_crd_kustomization "${generated_crd_dir}/kustomization.yaml"

	"${controller_gen}" \
		rbac:roleName=manager-role \
		paths="${RBAC_PATHS}" \
		output:rbac:artifacts:config="${generated_rbac_dir}"

	if [[ ! -f "${generated_rbac_dir}/role.yaml" ]]; then
		echo "no RBAC role.yaml was generated for group ${group}" >&2
		exit 1
	fi

	cat >"${generated_rbac_dir}/kustomization.yaml" <<'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- role.yaml
EOF
}

render_workdir() {
	local workdir

	workdir="${ROOT_DIR}/dist/.work/${group}"
	rm -rf "${workdir}"
	mkdir -p "${workdir}"

	cat >"${workdir}/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../../packages/${group}/install
EOF

	(
		cd "${workdir}"
		"${kustomize}" edit set image "controller=${controller_image}" >/dev/null
	)

	printf '%s\n' "${workdir}"
}

render_install() {
	local output_path=${OUT:-"${ROOT_DIR}/dist/packages/${group}/install.yaml"}
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
generate)
	generate_assets
	;;
render)
	generate_assets
	render_install
	;;
*)
	usage
	exit 1
	;;
esac
