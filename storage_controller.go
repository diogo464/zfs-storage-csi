package main

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

var _ csi.ControllerServer = (*StorageCsi)(nil)

// ControllerExpandVolume implements csi.ControllerServer.
func (*StorageCsi) ControllerExpandVolume(context.Context, *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	panic("unimplemented")
}

// ControllerGetCapabilities implements csi.ControllerServer.
func (*StorageCsi) ControllerGetCapabilities(context.Context, *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	capabilities := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_GET_CAPACITY,
		//csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		//csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		//csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_READONLY,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		//csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
		csi.ControllerServiceCapability_RPC_VOLUME_CONDITION,
		csi.ControllerServiceCapability_RPC_GET_VOLUME,
		csi.ControllerServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
		csi.ControllerServiceCapability_RPC_MODIFY_VOLUME,
	}

	res := &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{},
	}
	for _, capability := range capabilities {
		res.Capabilities = append(res.Capabilities, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: capability,
				},
			},
		})
	}

	return res, nil
}

// ControllerGetVolume implements csi.ControllerServer.
func (*StorageCsi) ControllerGetVolume(context.Context, *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	panic("unimplemented")
}

// ControllerModifyVolume implements csi.ControllerServer.
func (*StorageCsi) ControllerModifyVolume(context.Context, *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	panic("unimplemented")
}

// ControllerPublishVolume implements csi.ControllerServer.
func (*StorageCsi) ControllerPublishVolume(context.Context, *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	panic("unimplemented")
}

// ControllerUnpublishVolume implements csi.ControllerServer.
func (*StorageCsi) ControllerUnpublishVolume(context.Context, *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	panic("unimplemented")
}

// CreateSnapshot implements csi.ControllerServer.
func (*StorageCsi) CreateSnapshot(context.Context, *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	panic("unimplemented")
}

// CreateVolume implements csi.ControllerServer.
func (*StorageCsi) CreateVolume(context.Context, *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	panic("unimplemented")
}

// DeleteSnapshot implements csi.ControllerServer.
func (*StorageCsi) DeleteSnapshot(context.Context, *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	panic("unimplemented")
}

// DeleteVolume implements csi.ControllerServer.
func (*StorageCsi) DeleteVolume(context.Context, *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	panic("unimplemented")
}

// GetCapacity implements csi.ControllerServer.
func (*StorageCsi) GetCapacity(context.Context, *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	panic("unimplemented")
}

// ListSnapshots implements csi.ControllerServer.
func (*StorageCsi) ListSnapshots(context.Context, *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	panic("unimplemented")
}

// ListVolumes implements csi.ControllerServer.
func (*StorageCsi) ListVolumes(context.Context, *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	panic("unimplemented")
}

// ValidateVolumeCapabilities implements csi.ControllerServer.
func (*StorageCsi) ValidateVolumeCapabilities(context.Context, *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	panic("unimplemented")
}
