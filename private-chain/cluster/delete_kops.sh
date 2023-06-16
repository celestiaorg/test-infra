#!/bin/bash

err_report() {
    echo "Error on line $1"
}

trap 'err_report $LINENO' ERR

# =======================================================
kops delete cluster $CLUSTER_NAME --yes
# =======================================================
