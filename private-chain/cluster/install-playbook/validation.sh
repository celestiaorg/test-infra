#!/bin/bash

set -o errexit
set -o pipefail
set -e

# Validate required arguments
if [ -z "$CLUSTER_SPEC_TEMPLATE" ]
then
  echo -e "Please provider cluster spec template file as argument. For example: \`./01_install.sh cluster.yaml\`"
  exit 2
fi
if [ ! -f "$CLUSTER_SPEC_TEMPLATE" ]; then
    echo "Provided cluster spec file \"$1\" doesn't exist"
    exit 2
fi
if [ -z "$CLUSTER_NAME" ]
then
  echo -e "Environment variable CLUSTER_NAME must be set."
  exit 2
fi
if [ -z "$DEPLOYMENT_NAME" ]
then
  echo -e "Environment variable DEPLOYMENT_NAME must be set."
  exit 2
fi
if [ -z "$KOPS_STATE_STORE" ]
then
  echo -e "Environment variable KOPS_STATE_STORE must be set."
  exit 2
fi
if [ -z "$ZONE_A" ]
then
  echo -e "Environment variable ZONE_A must be set."
  exit 2
fi
if [ -z "$ZONE_B" ]
then
  echo -e "Environment variable ZONE_B must be set."
  exit 2
fi
if [ -z "$AWS_REGION" ]
then
  echo -e "Environment variable AWS_REGION must be set."
  exit 2
fi
if [ -z "$WORKER_NODE_TYPE" ]
then
  echo -e "Environment variable WORKER_NODE_TYPE must be set."
  exit 2
fi
if [ -z "$MASTER_NODE_TYPE" ]
then
  echo -e "Environment variable MASTER_NODE_TYPE must be set."
  exit 2
fi
if [ -z "$MAX_WORKER_NODES" ]
then
  echo -e "Environment variable MAX_WORKER_NODES must be set."
  exit 2
fi
if [ -z "$MIN_WORKER_NODES" ]
then
  echo -e "Environment variable MIN_WORKER_NODES must be set."
  exit 2
fi
if [ -z "$PUBKEY" ]
then
  echo -e "Environment variable PUBKEY must be set."
  exit 2
fi
if [ -z "$TEAM" ]
then
  echo -e "Environment variable TEAM must be set (used for cost-allocation purposes)."
  exit 2
fi
if [ -z "$PROJECT" ]
then
  echo -e "Environment variable PROJECT must be set (used for cost-allocation purposes)."
  exit 2
fi

# Default values for Optional params
if [ -z "$ULIMIT_NOFILE" ]
then
	export ULIMIT_NOFILE="1048576:1048576"
fi

