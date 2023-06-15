#!/bin/bash

kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0alertmanagerConfigCustomResourceDefinition.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0alertmanagerCustomResourceDefinition.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0podmonitorCustomResourceDefinition.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0probeCustomResourceDefinition.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0prometheusCustomResourceDefinition.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0prometheusruleCustomResourceDefinition.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0servicemonitorCustomResourceDefinition.yaml
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/v0.12.0/manifests/setup/0thanosrulerCustomResourceDefinition.yaml
