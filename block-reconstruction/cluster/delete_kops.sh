#!/bin/bash

err_report() {
    echo "Error on line $1"
}

trap 'err_report $LINENO' ERR

my_dir="$(dirname "$0")"
source "$my_dir/values.sh"

# =======================================================
kops delete cluster $CLUSTER_NAME --yes
# =======================================================
