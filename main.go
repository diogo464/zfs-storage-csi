package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
)

const (
	PLUGIN_NAME    = "csi.infra.d464.sh"
	PLUGIN_VERSION = "1.0.0"

	ENV_STORAGE_SSH_HOST    = "STORAGE_SSH_HOST"
	ENV_STORAGE_SSH_PORT    = "STORAGE_SSH_PORT"
	ENV_STORAGE_SSH_USER    = "STORAGE_SSH_USER"
	ENV_STORAGE_SSH_KEY     = "STORAGE_SSH_KEY"
	ENV_STORAGE_ZFS_SUDO    = "STORAGE_SSH_SUDO"
	ENV_STORAGE_ZFS_DATASET = "STORAGE_ZFS_DATASET"

	ZFS_PROPERTY_SHARENFS      = "sharenfs"
	ZFS_PROPERTY_SHARENFS_ON   = "on"
	ZFS_PROPERTY_NAMESPACE     = "k8s:namespace"
	ZFS_PROPERTY_PV            = "k8s:pv"
	ZFS_PROPERTY_PVC           = "k8s:pvc"
	ZFS_PROPERTY_DELETED       = "k8s:deleted"
	ZFS_PROPERTY_DELETED_TRUE  = "true"
	ZFS_PROPERTY_DELETED_FALSE = "false"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <controller|node>", os.Args[0])
	}

	listener, err := createListener()
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	zfsClient, err := createZfsClient()
	if err != nil {
		log.Fatalf("Error creating zfs client: %v", err)
	}

	mode := os.Args[1]
	if mode == "controller" {
		controller := &ControllerCsi{
			config: &ControllerConfig{
				ParentDataset: getEnvOrFail(ENV_STORAGE_ZFS_DATASET),
			},
			client: zfsClient,
		}
		csi.RegisterIdentityServer(grpcServer, controller)
		csi.RegisterControllerServer(grpcServer, controller)
	} else if mode == "node" {
		node := &NodeCsi{
			Config: &NodeConfig{
				NodeHostname:          getEnvOrFail("NODE_ID"),
				StorageHostname: getEnvOrFail(ENV_STORAGE_SSH_HOST),
				ParentDataset:   getEnvOrFail(ENV_STORAGE_ZFS_DATASET),
			},
			Client: zfsClient,
		}
		csi.RegisterIdentityServer(grpcServer, node)
		csi.RegisterNodeServer(grpcServer, node)
	} else {
		log.Fatalf("Invalid mode: %s", mode)
	}

	log.Printf("Listening for connections on address: %#v", listener.Addr())
	grpcServer.Serve(listener)
}

func createListener() (net.Listener, error) {
	endpoint := os.Getenv("CSI_ENDPOINT")

	if strings.HasPrefix(endpoint, "unix://") {
		sock := strings.TrimPrefix(endpoint, "unix://")
		if err := os.Remove(sock); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		lis, err := net.Listen("unix", sock)
		if err != nil {
			return nil, err
		}
		return lis, nil
	}

	if strings.HasPrefix(endpoint, "tcp://") {
		sock := strings.TrimPrefix(endpoint, "tcp://")
		lis, err := net.Listen("tcp", sock)
		if err != nil {
			return nil, err
		}
		return lis, nil
	}

	return nil, fmt.Errorf("invalid CSI_ENDPOINT: %s", endpoint)
}

func getEnvOrFail(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Missing required environment variable: %s", key)
	}
	return val
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func createSshClient() (*ssh.Client, error) {
	key := getEnvOrFail(ENV_STORAGE_SSH_KEY)
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		log.Printf("Error parsing key: %v", err)
		return nil, err
	}
	hostname := getEnvOrFail(ENV_STORAGE_SSH_HOST)
	port := getEnvOrDefault(ENV_STORAGE_SSH_PORT, "22")
	user := getEnvOrFail(ENV_STORAGE_SSH_USER)
	host := fmt.Sprintf("%s:%s", hostname, port)
	log.Printf("ssh dialing: %s@%s", user, host)
	return ssh.Dial("tcp", host, &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}

func createZfsClient() (*ZfsClient, error) {
	sshClient, err := createSshClient()
	if err != nil {
		return nil, err
	}
	sudo := getEnvOrFail(ENV_STORAGE_ZFS_SUDO) == "true"

	return &ZfsClient{
		sshClient: sshClient,
		sudo:      sudo,
	}, nil
}

func datasetNameFromVolumeName(parentDataset, volumeName string) string {
	if volumeName == "" {
		panic("volumeName must not be empty")
	}
	if parentDataset == "" {
		return volumeName
	}
	if parentDataset[len(parentDataset)-1] == '/' {
		return parentDataset + volumeName
	}
	return parentDataset + "/" + volumeName
}

func volumeNameFromDatasetName(datasetName string) string {
	parts := strings.Split(datasetName, "/")
	return parts[len(parts)-1]
}

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"strconv"
// 	"time"
//
// 	"golang.org/x/crypto/ssh"
// 	v1 "k8s.io/api/core/v1"
// 	storagev1 "k8s.io/api/storage/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/rest"
// 	"k8s.io/client-go/tools/clientcmd"
// 	storagehelpers "k8s.io/component-helpers/storage/volume"
// 	"sigs.k8s.io/sig-storage-lib-external-provisioner/v9/controller"
// )
//
// const (
// 	ENV_PROVISIONER_NAME     = "PROVISIONER_NAME"
// 	ENV_STORAGE_HOSTNAME     = "STORAGE_HOSTNAME"
// 	ENV_STORAGE_DATASET_BASE = "STORAGE_DATASET_BASE"
// 	ENV_STORAGE_SSH_KEY      = "STORAGE_SSH_KEY"
// 	ENV_STORAGE_SSH_PORT     = "STORAGE_SSH_PORT"
// 	ENV_STORAGE_SSH_USER     = "STORAGE_SSH_USER"
// 	ENV_STORAGE_SSH_SUDO     = "STORAGE_SSH_SUDO"
//
// 	STORAGE_CLASS_PARAMETER_LOCAL = "local"
//
// 	ZFS_PROPERTY_SHARENFS      = "sharenfs"
// 	ZFS_PROPERTY_SHARENFS_ON   = "on"
// 	ZFS_PROPERTY_NAMESPACE     = "k8s:namespace"
// 	ZFS_PROPERTY_PV            = "k8s:pv"
// 	ZFS_PROPERTY_PVC           = "k8s:pvc"
// 	ZFS_PROPERTY_DELETED       = "k8s:deleted"
// 	ZFS_PROPERTY_DELETED_TRUE  = "true"
// 	ZFS_PROPERTY_DELETED_FALSE = "false"
// )
//
// var _ controller.Provisioner = (*StorageProvisioner)(nil)
//
// type storageProvisionerConfig struct {
// 	provisionerName string
// 	storageHostname string
// 	sshKey          string
// 	sshPort         int
// 	sshUser         string
// 	sshSudo         bool
// 	zfsDatasetBase  string
// }
//
// type StorageProvisioner struct {
// 	config storageProvisionerConfig
// 	client kubernetes.Interface
// }
//
// // Delete implements controller.Provisioner.
// func (sp *StorageProvisioner) Delete(ctx context.Context, pv *v1.PersistentVolume) error {
// 	log.Printf("Deleting volume: %s", pv.Name)
//
// 	client, err := sp.createZfsClient()
// 	if err != nil {
// 		log.Printf("Error creating zfs client: %v", err)
// 		return err
// 	}
//
// 	datasetName := sp.datasetNameFromVolumeName(pv.Name)
// 	exists, err := client.DatasetExists(datasetName)
// 	if err != nil {
// 		return err
// 	}
//
// 	if exists {
// 		log.Printf("Dataset exists, deleting: %s", datasetName)
// 		return client.UpdateProperty(datasetName, ZFS_PROPERTY_DELETED, ZFS_PROPERTY_DELETED_TRUE)
// 	} else {
// 		log.Printf("Dataset does not exist, skipping deletion: %s", datasetName)
// 		return nil
// 	}
// }
//
// // Provision implements controller.Provisioner.
// func (sp *StorageProvisioner) Provision(ctx context.Context, po controller.ProvisionOptions) (*v1.PersistentVolume, controller.ProvisioningState, error) {
// 	log.Printf("Provisioning volume: %s", po.PVName)
// 	client, err := sp.createZfsClient()
// 	if err != nil {
// 		return nil, controller.ProvisioningNoChange, err
// 	}
//
// 	datasetName := sp.datasetNameFromVolumeName(po.PVName)
// 	properties := map[string]string{
// 		ZFS_PROPERTY_NAMESPACE: po.PVC.Namespace,
// 		ZFS_PROPERTY_PV:        po.PVName,
// 		ZFS_PROPERTY_PVC:       po.PVC.Name,
// 		ZFS_PROPERTY_DELETED:   ZFS_PROPERTY_DELETED_FALSE,
// 		ZFS_PROPERTY_SHARENFS:  ZFS_PROPERTY_SHARENFS_ON,
// 	}
// 	if err := client.CreateDatasetIfNotExists(datasetName, properties); err != nil {
// 		log.Printf("Error creating dataset: %v", err)
// 		return nil, controller.ProvisioningNoChange, err
// 	}
// 	if err := client.ShareDataset(datasetName); err != nil {
// 		log.Printf("Error sharing dataset: %v", err)
// 		return nil, controller.ProvisioningNoChange, err
// 	}
// 	if err := client.ChmodDataset(datasetName, "777"); err != nil {
// 		log.Printf("Error chmoding dataset: %v", err)
// 		return nil, controller.ProvisioningNoChange, err
// 	}
//
// 	volumeSize := po.PVC.Spec.Resources.Requests[v1.ResourceStorage]
// 	volumeSizeBytes := volumeSize.Value()
// 	if err := client.SetDatasetQuota(datasetName, volumeSizeBytes); err != nil {
// 		log.Printf("Error setting quota: %v", err)
// 		return nil, controller.ProvisioningNoChange, err
// 	}
//
// 	source, affinity := sp.sourceAndAffinity(client, &po)
// 	pv := &v1.PersistentVolume{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: po.PVName,
// 		},
// 		Spec: v1.PersistentVolumeSpec{
// 			Capacity: v1.ResourceList{
// 				v1.ResourceStorage: po.PVC.Spec.Resources.Requests[v1.ResourceStorage],
// 			},
// 			PersistentVolumeSource:        source,
// 			AccessModes:                   po.PVC.Spec.AccessModes,
// 			PersistentVolumeReclaimPolicy: *po.StorageClass.ReclaimPolicy,
// 			MountOptions:                  po.StorageClass.MountOptions,
// 			NodeAffinity:                  affinity,
// 		},
// 	}
//
// 	return pv, controller.ProvisioningFinished, nil
// }
//
// func (sp *StorageProvisioner) sourceAndAffinity(client *ZfsClient, po *controller.ProvisionOptions) (v1.PersistentVolumeSource, *v1.VolumeNodeAffinity) {
// 	datasetName := sp.datasetNameFromVolumeName(po.PVName)
// 	mountpoint, err := client.GetDatasetMountpoint(datasetName)
// 	if err != nil {
// 		return v1.PersistentVolumeSource{}, &v1.VolumeNodeAffinity{}
// 	}
//
// 	if sp.isProvisionLocal(po) {
// 		source := v1.PersistentVolumeSource{
// 			HostPath: &v1.HostPathVolumeSource{
// 				Path: mountpoint,
// 			},
// 		}
// 		affinity := &v1.VolumeNodeAffinity{
// 			Required: &v1.NodeSelector{
// 				NodeSelectorTerms: []v1.NodeSelectorTerm{
// 					{
// 						MatchExpressions: []v1.NodeSelectorRequirement{
// 							{
// 								Key:      "kubernetes.io/hostname",
// 								Operator: v1.NodeSelectorOpIn,
// 								Values:   []string{sp.config.storageHostname},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		}
// 		return source, affinity
// 	} else {
// 		source := v1.PersistentVolumeSource{
// 			NFS: &v1.NFSVolumeSource{
// 				Server:   sp.config.storageHostname,
// 				Path:     mountpoint,
// 				ReadOnly: false,
// 			},
// 		}
// 		return source, nil
// 	}
// }
//
// func (sp *StorageProvisioner) isProvisionLocal(po *controller.ProvisionOptions) bool {
// 	return po.StorageClass.Parameters[STORAGE_CLASS_PARAMETER_LOCAL] == "true"
// }
//
//
//
// // getClassForVolume returns StorageClass.
// // https://github.com/kubernetes-sigs/nfs-subdir-external-provisioner/blob/master/cmd/nfs-subdir-external-provisioner/provisioner.go#L192
// func (sp *StorageProvisioner) getClassForVolume(ctx context.Context, pv *v1.PersistentVolume) (*storagev1.StorageClass, error) {
// 	if sp.client == nil {
// 		return nil, fmt.Errorf("cannot get kube client")
// 	}
// 	className := storagehelpers.GetPersistentVolumeClass(pv)
// 	if className == "" {
// 		return nil, fmt.Errorf("volume has no storage class")
// 	}
// 	class, err := sp.client.StorageV1().StorageClasses().Get(ctx, className, metav1.GetOptions{})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return class, nil
// }
//
// func main() {
// 	config, err := getKubeConfig()
// 	if err != nil {
// 		log.Fatalf("Error getting kubeconfig: %v", err)
// 	}
//
// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		log.Fatalf("Error getting clientset: %v", err)
// 	}
//
// 	provisionerName := getEnvOrFail(ENV_PROVISIONER_NAME)
//
// 	ctrl := controller.NewProvisionController(clientset, provisionerName, &StorageProvisioner{
// 		config: configFromEnv(),
// 		client: clientset,
// 	}, controller.LeaderElection(false), controller.ResyncPeriod(time.Duration(15)*time.Second))
//
// 	log.Println("Starting provisioner")
// 	ctrl.Run(context.Background())
// }
//
// func getKubeConfig() (*rest.Config, error) {
// 	kubeconfig := os.Getenv("KUBECONFIG")
// 	if kubeconfig != "" {
// 		log.Printf("Using kubeconfig: %s", kubeconfig)
// 		return clientcmd.BuildConfigFromFlags("", kubeconfig)
// 	} else {
// 		log.Println("Using in cluster config")
// 		return rest.InClusterConfig()
// 	}
// }
//
// func getEnvOrFail(key string) string {
// 	value := os.Getenv(key)
// 	if value == "" {
// 		log.Fatalf("Missing required environment variable: %s", key)
// 	}
// 	return value
// }
//
//
// func configFromEnv() storageProvisionerConfig {
// 	hostname := getEnvOrFail(ENV_STORAGE_HOSTNAME)
// 	port := getEnvOrDefault(ENV_STORAGE_SSH_PORT, "22")
// 	user := getEnvOrFail(ENV_STORAGE_SSH_USER)
// 	sudo := getEnvOrDefault(ENV_STORAGE_SSH_SUDO, "false")
//
// 	portInt, err := strconv.Atoi(port)
// 	if err != nil {
// 		log.Fatalf("Error parsing port: %v", err)
// 	}
//
// 	sudoBool := false
// 	if sudo == "true" {
// 		sudoBool = true
// 	}
//
// 	return storageProvisionerConfig{
// 		provisionerName: getEnvOrFail(ENV_PROVISIONER_NAME),
// 		storageHostname: hostname,
// 		sshKey:          getEnvOrFail(ENV_STORAGE_SSH_KEY),
// 		sshPort:         portInt,
// 		sshUser:         user,
// 		sshSudo:         sudoBool,
// 		zfsDatasetBase:  getEnvOrFail(ENV_STORAGE_DATASET_BASE),
// 	}
// }
