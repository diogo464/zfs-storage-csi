# storage-csi

the csi storage provisioner can provision volumes by creating zfs datasets on a node running zfs.
the volumes can be served over nfs or locally depending on which node the pod is scheduled in.
the provisioner needs ssh access to node running zfs.

## example installation
this example installation assumes the node running zfs is `citadel` and it creates a storage class named `blackmesa`.

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: storage
resources:
  - https://git.d464.sh/infra/storage-csi//deploy?ref=main
  - storageclass.yaml
  - secret.yaml
patches:
  - path: secret.yaml
```

`storageclass.yaml`
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: blackmesa
provisioner: infra.d464.sh
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
```

`secret.yaml`
```yaml
apiVersion: v1
kind: Secret
metadata:
    name: storage-csi
stringData:
    STORAGE_HOST: "citadel"
    STORAGE_SSH_PORT: "22"
    STORAGE_SSH_USER: "core"
    STORAGE_SSH_KEY: |
      <ssh private key>
    STORAGE_SSH_SUDO: "true"
    STORAGE_ZFS_DATASET: "blackmesa/csi"
```