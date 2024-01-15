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
	ENV_STORAGE_HOST        = "STORAGE_HOST"
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
				NodeHostname:    getEnvOrFail("NODE_ID"),
				StorageHostname: getEnvOrFail(ENV_STORAGE_HOST),
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
	hostname := getEnvOrFail(ENV_STORAGE_HOST)
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
