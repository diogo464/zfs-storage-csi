package main

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

var _ csi.NodeServer = (*StorageCsi)(nil)

// NodeExpandVolume implements csi.NodeServer.
func (*StorageCsi) NodeExpandVolume(context.Context, *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	panic("unimplemented")
}

// NodeGetCapabilities implements csi.NodeServer.
func (*StorageCsi) NodeGetCapabilities(context.Context, *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	panic("unimplemented")
}

// NodeGetInfo implements csi.NodeServer.
func (*StorageCsi) NodeGetInfo(context.Context, *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	panic("unimplemented")
}

// NodeGetVolumeStats implements csi.NodeServer.
func (*StorageCsi) NodeGetVolumeStats(context.Context, *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	panic("unimplemented")
}

// NodePublishVolume implements csi.NodeServer.
func (*StorageCsi) NodePublishVolume(context.Context, *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	panic("unimplemented")
}

// NodeStageVolume implements csi.NodeServer.
func (*StorageCsi) NodeStageVolume(context.Context, *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	panic("unimplemented")
}

// NodeUnpublishVolume implements csi.NodeServer.
func (*StorageCsi) NodeUnpublishVolume(context.Context, *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	panic("unimplemented")
}

// NodeUnstageVolume implements csi.NodeServer.
func (*StorageCsi) NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	panic("unimplemented")
}
