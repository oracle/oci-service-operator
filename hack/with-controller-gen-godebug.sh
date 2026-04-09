#!/usr/bin/env bash

set -euo pipefail

compat_setting=${CONTROLLER_GEN_GODEBUG:-gotypesalias=0}
current_godebug=${GODEBUG:-}

# Preserve any caller-supplied GODEBUG settings while forcing controller-gen's
# go/types alias behavior back to the pre-Go-1.26 mode by default.
if [[ -n "${compat_setting}" ]]; then
	setting_name=${compat_setting%%=*}
	if [[ ",${current_godebug}," != *",${setting_name}="* ]]; then
		if [[ -n "${current_godebug}" ]]; then
			current_godebug="${current_godebug},${compat_setting}"
		else
			current_godebug="${compat_setting}"
		fi
	fi
	export GODEBUG="${current_godebug}"
fi

exec "$@"
