package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ csi.IdentityServer = (*ControllerCsi)(nil)
var _ csi.ControllerServer = (*ControllerCsi)(nil)

type ControllerConfig struct {
	ParentDataset string
}

type ControllerCsi struct {
	config *ControllerConfig
	client *ZfsClient
}

// GetPluginCapabilities implements csi.IdentityServer.
func (*ControllerCsi) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	res := &csi.GetPluginCapabilitiesResponse{
		Capabilities: PLUGIN_CAPABILITIES,
	}
	log.Printf("GetPluginCapabilities: %v", res)
	return res, nil
}

// GetPluginInfo implements csi.IdentityServer.
func (*ControllerCsi) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	res := &csi.GetPluginInfoResponse{
		Name:          PLUGIN_NAME,
		VendorVersion: PLUGIN_VERSION,
	}
	log.Printf("GetPluginInfo: %v", res)
	return res, nil
}

// Probe implements csi.IdentityServer.
func (*ControllerCsi) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	res := &csi.ProbeResponse{Ready: wrapperspb.Bool(true)}
	// log.Printf("Probe: %v", res)
	return res, nil
}

// ControllerExpandVolume implements csi.ControllerServer.
func (n *ControllerCsi) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	log.Printf("ControllerExpandVolume: %v", req)
	dataset, err := findExistingDatasetByVolumeId(n.client, req.VolumeId)
	if err != nil {
		return nil, err
	}
	if req.CapacityRange == nil {
		return nil, status.Error(codes.InvalidArgument, "capacity range must be specified")
	}
	if req.CapacityRange.RequiredBytes == 0 {
		return nil, status.Error(codes.InvalidArgument, "required bytes must be specified")
	}
	capacity := int64(req.CapacityRange.RequiredBytes)
	if err := n.client.SetDatasetQuota(dataset, capacity); err != nil {
		log.Printf("Error setting quota: %v", err)
		return nil, err
	}
	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         capacity,
		NodeExpansionRequired: false,
	}, nil
}

// ControllerGetCapabilities implements csi.ControllerServer.
func (*ControllerCsi) ControllerGetCapabilities(context.Context, *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	capabilities := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		//csi.ControllerServiceCapability_RPC_GET_CAPACITY,
		//csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		//csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		//csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
		//csi.ControllerServiceCapability_RPC_PUBLISH_READONLY,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		//csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
		//csi.ControllerServiceCapability_RPC_VOLUME_CONDITION,
		//csi.ControllerServiceCapability_RPC_GET_VOLUME,
		csi.ControllerServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
		//csi.ControllerServiceCapability_RPC_MODIFY_VOLUME,
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

	log.Printf("ControllerGetCapabilities: %v", res)
	return res, nil
}

// ControllerGetVolume implements csi.ControllerServer.
func (*ControllerCsi) ControllerGetVolume(context.Context, *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "get volume not supported")
}

// ControllerModifyVolume implements csi.ControllerServer.
func (*ControllerCsi) ControllerModifyVolume(context.Context, *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "modify volume not supported")
}

// ControllerPublishVolume implements csi.ControllerServer.
func (c *ControllerCsi) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	// most of the work is done in NodePublishVolume
	// in here we just make sure the dataset is shared
	log.Printf("ControllerPublishVolume: %v", req)
	dataset, err := findExistingDatasetByVolumeId(c.client, req.VolumeId)
	if err != nil {
		log.Printf("Error finding dataset by volume id: %v", err)
		return nil, err
	}

	if err := c.client.ShareDataset(dataset); err != nil {
		log.Printf("Error sharing dataset: %v", err)
		return nil, err
	}
	return &csi.ControllerPublishVolumeResponse{}, nil
}

// ControllerUnpublishVolume implements csi.ControllerServer.
func (*ControllerCsi) ControllerUnpublishVolume(context.Context, *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	// nothing to do here
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

// CreateSnapshot implements csi.ControllerServer.
func (*ControllerCsi) CreateSnapshot(context.Context, *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "create snapshot not supported")
}

// CreateVolume implements csi.ControllerServer.
func (c *ControllerCsi) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	log.Printf("CreateVolume: %v", req)

	if req.CapacityRange == nil {
		return nil, status.Error(codes.InvalidArgument, "capacity range must be specified")
	}
	if req.CapacityRange.RequiredBytes == 0 {
		return nil, status.Error(codes.InvalidArgument, "required bytes must be specified")
	}
	// TODO: validate VolumeCapabilities
	if req.VolumeContentSource != nil {
		return nil, status.Error(codes.InvalidArgument, "volume content source not supported")
	}
	if req.AccessibilityRequirements != nil {
		return nil, status.Error(codes.InvalidArgument, "accessibility requirements not supported")
	}
	if req.MutableParameters != nil {
		return nil, status.Error(codes.InvalidArgument, "mutable parameters not supported")
	}
	if req.Parameters == nil {
		return nil, status.Error(codes.InvalidArgument, "parameters must be specified")
	}

	namespace := req.Parameters["csi.storage.k8s.io/pvc/namespace"]
	if namespace == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace must be specified")
	}

	pvc := req.Parameters["csi.storage.k8s.io/pvc/name"]
	if pvc == "" {
		return nil, status.Error(codes.InvalidArgument, "pvc must be specified")
	}

	pv := req.Parameters["csi.storage.k8s.io/pv/name"]
	if pv == "" {
		return nil, status.Error(codes.InvalidArgument, "pv must be specified")
	}

	// we use a search by properties for backwards compatibility
	// the name of the dataset has changed over time but the properties have not
	// and they containing information to indentify the dataset.
	zfsSearchProperties := map[string]string{
		ZFS_PROPERTY_NAMESPACE: namespace,
		ZFS_PROPERTY_PVC:       pvc,
		ZFS_PROPERTY_DELETED:   ZFS_PROPERTY_DELETED_FALSE,
	}

	zfsProperties := map[string]string{
		ZFS_PROPERTY_SHARENFS:  ZFS_PROPERTY_SHARENFS_ON,
		ZFS_PROPERTY_NAMESPACE: namespace,
		ZFS_PROPERTY_PV:        pv,
		ZFS_PROPERTY_PVC:       pvc,
		ZFS_PROPERTY_DELETED:   ZFS_PROPERTY_DELETED_FALSE,
	}

	datasetName := createDatasetName(c.config.ParentDataset, namespace, pvc)
	log.Printf("searching for dataset with properties: %v", zfsSearchProperties)
	foundDataset, err := c.client.FindDatasetByProperties(zfsSearchProperties)
	if err != nil {
		return nil, err
	}

	if foundDataset != "" {
		log.Printf("found an existing dataset: %s", foundDataset)
		if foundDataset != datasetName {
			log.Printf("found an existing dataset with a different name: %s", foundDataset)
			if err := c.client.RenameDataset(foundDataset, datasetName); err != nil {
				log.Printf("Error renaming dataset: %v", err)
				return nil, err
			}
		}

		log.Printf("updating properties of existing dataset: %s", datasetName)
		c.client.UpdateProperties(datasetName, zfsProperties)
	}

	if err := c.client.CreateDatasetIfNotExists(datasetName, zfsProperties); err != nil {
		log.Printf("Error creating dataset: %v", err)
		return nil, err
	}

	if err := c.client.ChmodDataset(datasetName, "777"); err != nil {
		log.Printf("Error chmoding dataset: %v", err)
		return nil, err
	}

	if err := c.client.SetDatasetQuota(datasetName, int64(req.CapacityRange.RequiredBytes)); err != nil {
		log.Printf("Error setting quota: %v", err)
		return nil, err
	}

	if err := c.client.ShareDataset(datasetName); err != nil {
		log.Printf("Error sharing dataset: %v", err)
		return nil, err
	}

	res := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: req.CapacityRange.RequiredBytes,
			VolumeId:      req.Name,
		},
	}
	log.Printf("CreateVolume: %v", res)
	return res, nil
}

// DeleteSnapshot implements csi.ControllerServer.
func (*ControllerCsi) DeleteSnapshot(context.Context, *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "delete snapshot not supported")
}

// DeleteVolume implements csi.ControllerServer.
func (c *ControllerCsi) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	log.Printf("DeleteVolume: %v", req)
	log.Printf("Deleting volume: %s", req.VolumeId)

	dataset, err := findExistingDatasetByVolumeId(c.client, req.VolumeId)
	if err != nil {
		log.Printf("Error finding dataset by volume id: %v", err)
		return nil, err
	}

	exists, err := c.client.DatasetExists(dataset)
	if err != nil {
		log.Printf("Error checking if dataset exists: %v", err)
		return nil, err
	}

	timestamp := time.Now().Unix()
	deletedDatasetName := fmt.Sprintf("%s-%d", dataset, timestamp)

	if exists {
		log.Printf("Dataset exists, deleting: %s", dataset)
		if err := c.client.UpdateProperty(dataset, ZFS_PROPERTY_DELETED, ZFS_PROPERTY_DELETED_TRUE); err != nil {
			log.Printf("Error setting deleted property: %v", err)
			return nil, err
		}
		if err := c.client.RenameDataset(dataset, deletedDatasetName); err != nil {
			log.Printf("Error renaming dataset: %v", err)
			return nil, err
		}
	} else {
		log.Printf("Dataset does not exist, skipping deletion: %s", dataset)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

// GetCapacity implements csi.ControllerServer.
func (*ControllerCsi) GetCapacity(context.Context, *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "get capacity not supported")
}

// ListSnapshots implements csi.ControllerServer.
func (*ControllerCsi) ListSnapshots(context.Context, *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "list snapshots not supported")
}

// ListVolumes implements csi.ControllerServer.
func (c *ControllerCsi) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	if req.MaxEntries != 0 {
		return nil, status.Error(codes.InvalidArgument, "max entries not supported")
	}
	if req.StartingToken != "" {
		return nil, status.Error(codes.InvalidArgument, "starting token not supported")
	}

	volumes := []*csi.Volume{}
	if datasets, err := c.client.ListChildDatasets(c.config.ParentDataset); err == nil {
		for _, dataset := range datasets {
			if dataset.quota == nil {
				// quota should never be nil since we require it when creating a dataset
				return nil, status.Error(codes.Internal, "dataset quota is nil")
			}

			deleted, err := c.client.GetProperty(dataset.name, ZFS_PROPERTY_DELETED)
			if err != nil {
				log.Printf("Error getting deleted property: %v", err)
				return nil, err
			}

			if deleted != ZFS_PROPERTY_DELETED_FALSE {
				log.Printf("Dataset is marked as deleted: %s", dataset.name)
				continue
			}

			volumeId, err := c.client.GetProperty(dataset.name, ZFS_PROPERTY_PV)
			if err != nil {
				log.Printf("Error getting pv property: %v", err)
				return nil, err
			}

			volumes = append(volumes, &csi.Volume{
				CapacityBytes: int64(*dataset.quota),
				VolumeId:      volumeId,
			})
		}
	} else {
		log.Printf("Error listing datasets: %v", err)
		return nil, err
	}

	entries := []*csi.ListVolumesResponse_Entry{}
	for _, volume := range volumes {
		entries = append(entries, &csi.ListVolumesResponse_Entry{
			Volume: volume,
		})
	}
	res := &csi.ListVolumesResponse{
		Entries: entries,
	}
	log.Printf("ListVolumes: %v", res)
	return res, nil
}

// ValidateVolumeCapabilities implements csi.ControllerServer.
func (*ControllerCsi) ValidateVolumeCapabilities(context.Context, *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "validate volume capabilities not supported")
}
