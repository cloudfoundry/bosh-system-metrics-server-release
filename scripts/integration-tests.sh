#!/bin/bash

set -eo pipefail

time=$(date +%s%N)
cf install-plugin "log-cache" -f
sleep 120
cf tail bosh-system-metrics-forwarder -n 1000 --json --start-time="${time}" |
    jq -r '.batch[]?.gauge.metrics | keys[]?' |
    sort -u > /tmp/actual-metrics

cat > /tmp/expected-metrics << EOF
system.cpu.sys
system.cpu.user
system.cpu.wait
system.disk.ephemeral.inode_percent
system.disk.ephemeral.percent
system.disk.persistent.inode_percent
system.disk.persistent.percent
system.disk.system.inode_percent
system.disk.system.percent
system.healthy
system.load.1m
system.mem.kb
system.mem.percent
system.swap.kb
system.swap.percent
EOF

diff /tmp/expected-metrics /tmp/actual-metrics
