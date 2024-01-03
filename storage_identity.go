package main

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes/wrappers"
)

var _ csi.IdentityServer = (*StorageCsi)(nil)

// GetPluginCapabilities implements csi.IdentityServer.
func (*StorageCsi) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	res := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_VolumeExpansion_{
					VolumeExpansion: &csi.PluginCapability_VolumeExpansion{
						Type: csi.PluginCapability_VolumeExpansion_ONLINE,
					},
				},
			},
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}
	return res, nil
}

// GetPluginInfo implements csi.IdentityServer.
func (*StorageCsi) GetPluginInfo(context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	res := &csi.GetPluginInfoResponse{
		Name:          "infra.d464.sh",
		VendorVersion: "1.0.0",
		Manifest:      map[string]string{},
	}
	return res, nil
}

// Probe implements csi.IdentityServer.
func (*StorageCsi) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	res := &csi.ProbeResponse{
		Ready: &wrappers.BoolValue{Value: true},
	}
	return res, nil
}
