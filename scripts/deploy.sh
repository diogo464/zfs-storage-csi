#!/bin/sh
PUSH=1 scripts/build.sh || exit 1
kubectl delete -n storage-csi pods/csi-hostpathplugin-0
kubectl rollout restart -n storage-csi statefulset/csi-hostpathplugin
kubectl rollout restart -n storage-csi daemonset/storage-csi
