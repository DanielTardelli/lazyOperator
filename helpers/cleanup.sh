#!/bin/sh

kubectl delete deployment lazy
kubectl delete clusterrole cluster-admin-full-access
kubectl delete serviceaccount operator-sa
kubectl delete clusterrolebinding operator-sa-cluster-admin-binding
