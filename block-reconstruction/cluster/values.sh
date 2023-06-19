#!/bin/bash
export CLUSTER_NAME=testground.k8s.local
export DEPLOYMENT_NAME=testground.k8s.local

export WORKER_NODE_TYPE=c5a.8xlarge # 8CPU/16RAM
export MASTER_NODE_TYPE=c5a.8xlarge # 8CPU/16RAM
export MIN_WORKER_NODES=10
export MAX_WORKER_NODES=100

export TEAM=devops
export PROJECT=devops
export AWS_REGION=eu-west-1
export KOPS_STATE_STORE=s3://testground-terraform-state
export ZONE_A=eu-west-1a
export ZONE_B=eu-west-1b
#export PUBKEY=
export AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id)
export AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key)
