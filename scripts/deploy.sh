#!/bin/sh
PUSH=1 scripts/build.sh || exit 1
kubectl delete -n storage-csi pods/storage-csi-0
kubectl rollout restart -n storage-csi statefulset/storage-csi
kubectl rollout restart -n storage-csi daemonset/storage-csi
