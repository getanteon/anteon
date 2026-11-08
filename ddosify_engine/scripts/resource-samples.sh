#!/bin/sh
# Prints server resource samples for `ddosify report --resources`, as CSV of
# timestamp,source,metric,value, until it is stopped. Run it beside the load test:
#
#   scripts/resource-samples.sh docker  [INTERVAL_S]          > docker.csv  # every container (docker stats)
#   scripts/resource-samples.sh kubectl [INTERVAL_S] [ARGS]   > pods.csv    # kubectl top pods ARGS, such as -n NS -l app=x
#   KUBECTL="oc adm" scripts/resource-samples.sh kubectl 15 -n NS > pods.csv  # OpenShift
#   scripts/resource-samples.sh nvidia  [INTERVAL_MS]         > gpus.csv    # every GPU (nvidia-smi), as gpu0, gpu1, ...
#
# Timestamps are this machine's clock in Unix seconds: run it where the load generator runs, or on
# a machine whose clock agrees with it. COUNT=N stops after N rounds (nvidia: N lines).
set -eu

mode=${1:-}
[ $# -gt 0 ] && shift
count=${COUNT:-0}

# round runs its command every interval seconds until COUNT rounds are done.
round() {
	interval=$1
	shift
	n=0
	while :; do
		"$@"
		n=$((n + 1))
		if [ "$count" -gt 0 ] && [ "$n" -ge "$count" ]; then
			break
		fi
		sleep "$interval"
	done
}

docker_once() {
	docker stats --no-stream --format '{{.Name}},{{.CPUPerc}},{{.MemUsage}}' |
		awk -F, -v ts="$(date +%s)" '{ split($3, m, " / "); print ts "," $1 ",cpu_pct," $2; print ts "," $1 ",mem_bytes," m[1] }'
}

kubectl_once() {
	# pods: NAME CPU MEMORY; with --containers: POD NAME CPU MEMORY
	${KUBECTL:-kubectl} top pods --no-headers "$@" |
		awk -v ts="$(date +%s)" 'NF == 3 { s = $1; c = $2; m = $3 } NF == 4 { s = $1 "/" $2; c = $3; m = $4 } NF == 3 || NF == 4 { print ts "," s ",cpu_cores," c; print ts "," s ",mem_bytes," m }'
}

case "$mode" in
docker | kubectl | nvidia) echo "timestamp,source,metric,value" ;;
*)
	sed -n '2,11s/^# \{0,1\}//p' "$0" >&2
	exit 2
	;;
esac
case "$mode" in
docker)
	round "${1:-2}" docker_once
	;;
kubectl)
	interval=${1:-15}
	[ $# -gt 0 ] && shift
	round "$interval" kubectl_once "$@"
	;;
nvidia)
	n=0
	nvidia-smi --query-gpu=index,utilization.gpu,memory.used --format=csv,noheader,nounits -lms "${1:-1000}" |
		while IFS=', ' read -r i u m; do
			ts=$(date +%s)
			echo "$ts,gpu$i,gpu_util_pct,$u"
			echo "$ts,gpu$i,gpu_mem_bytes,${m}Mi"
			n=$((n + 1))
			if [ "$count" -gt 0 ] && [ "$n" -ge "$count" ]; then
				break
			fi
		done
	;;
esac
