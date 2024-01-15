package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"syscall"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ csi.IdentityServer = (*NodeCsi)(nil)
var _ csi.NodeServer = (*NodeCsi)(nil)

type NodeConfig struct {
	NodeHostname    string
	StorageHostname string
	ParentDataset   string
}

type NodeCsi struct {
	Config *NodeConfig
	Client *ZfsClient
}

// GetPluginCapabilities implements csi.IdentityServer.
func (*NodeCsi) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	res := &csi.GetPluginCapabilitiesResponse{
		Capabilities: PLUGIN_CAPABILITIES,
	}
	log.Printf("GetPluginCapabilities: %v", res)
	return res, nil
}

// GetPluginInfo implements csi.IdentityServer.
func (*NodeCsi) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	res := &csi.GetPluginInfoResponse{
		Name:          PLUGIN_NAME,
		VendorVersion: PLUGIN_VERSION,
	}
	log.Printf("GetPluginInfo: %v", res)
	return res, nil
}

// Probe implements csi.IdentityServer.
func (*NodeCsi) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	res := &csi.ProbeResponse{Ready: wrapperspb.Bool(true)}
	// log.Printf("Probe: %v", res)
	return res, nil
}

// NodeExpandVolume implements csi.NodeServer.
func (*NodeCsi) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	// this is implemented on the controller side
	return nil, fmt.Errorf("expansion not supported")
}

// NodeGetCapabilities implements csi.NodeServer.
func (n *NodeCsi) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	res := &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			//{
			//	Type: &csi.NodeServiceCapability_Rpc{
			//		Rpc: &csi.NodeServiceCapability_RPC{
			//			Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			//		},
			//	},
			//},
		},
	}
	// log.Printf("NodeGetCapabilities: %v", res)
	return res, nil
}

// NodeGetInfo implements csi.NodeServer.
func (n *NodeCsi) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	res := &csi.NodeGetInfoResponse{
		NodeId: n.Config.NodeHostname,
	}
	log.Printf("NodeGetInfo: %v", res)
	return res, nil
}

// NodeGetVolumeStats implements csi.NodeServer.
func (*NodeCsi) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, fmt.Errorf("stats not supported")
}

// NodePublishVolume implements csi.NodeServer.
func (n *NodeCsi) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	log.Printf("NodeStageVolume: %v", req)

	if err := os.MkdirAll(req.TargetPath, 0755); err != nil {
		log.Printf("Error creating target path %s: %v", req.TargetPath, err)
		return nil, err
	}

	if n.Config.StorageHostname == n.Config.NodeHostname {
		log.Printf("Node is storage node, mounting locally")
		mountpoint := path.Join("/dataset", req.VolumeId)
		if err := n.nodePublishVolumeLocal(ctx, mountpoint, req.TargetPath); err != nil {
			return nil, err
		}
	} else {
		log.Printf("Node is not storage node, mounting via NFS")

		dataset := datasetNameFromVolumeName(n.Config.ParentDataset, req.VolumeId)
		mountpoint, err := n.Client.GetDatasetMountpoint(dataset)
		if err != nil {
			log.Printf("Error getting mountpoint for dataset %s: %v", dataset, err)
			return nil, err
		}

		if err := n.nodePublishVolumeNfs(ctx, mountpoint, req.TargetPath); err != nil {
			return nil, err
		}
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeStageVolume implements csi.NodeServer.
func (n *NodeCsi) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, fmt.Errorf("staging not supported")
}

// NodeUnpublishVolume implements csi.NodeServer.
func (n *NodeCsi) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	log.Printf("NodeUnpublishVolume: %v", req)
	if err := syscall.Unmount(req.TargetPath, 0); err != nil && !os.IsNotExist(err) {
		log.Printf("Error unmounting %s: %v", req.TargetPath, err)
		return nil, err
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeUnstageVolume implements csi.NodeServer.
func (n *NodeCsi) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, fmt.Errorf("unstaging not supported")
}

func (n *NodeCsi) nodePublishVolumeLocal(ctx context.Context, mountpoint, target string) error {
	log.Printf("Mounting %s at %s", mountpoint, target)
	if err := syscall.Mount(mountpoint, target, "", syscall.MS_BIND, ""); err != nil {
		log.Printf("Error mounting %s: %v", mountpoint, err)
		return err
	}
	return nil
}

func (n *NodeCsi) nodePublishVolumeNfs(ctx context.Context, mountpoint, target string) error {
	ips, err := net.LookupHost(n.Config.StorageHostname)
	if err != nil {
		log.Printf("Error looking up hostname %s: %v", n.Config.StorageHostname, err)
		return err
	}
	if len(ips) == 0 {
		log.Printf("No IPs found for hostname %s", n.Config.StorageHostname)
		return fmt.Errorf("no IPs found for hostname %s", n.Config.StorageHostname)
	}
	ip := ips[0]

	// https://stackoverflow.com/questions/28350912/nfs-mount-system-call-in-linux
	source := fmt.Sprintf(":%s", mountpoint)
	options := fmt.Sprintf("addr=%v", ip)
	log.Printf("Mounting %s at %s with options %s", source, target, options)
	if err := syscall.Mount(source, target, "nfs4", 0, options); err != nil {
		log.Printf("Error mounting %s: %v", source, err)
		return err
	}
	return nil
}
