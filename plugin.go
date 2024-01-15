package main

import "github.com/container-storage-interface/spec/lib/go/csi"

const (
	PLUGIN_NAME    = "csi.infra.d464.sh"
	PLUGIN_VERSION = "1.0.0"
)

var PLUGIN_CAPABILITIES = []*csi.PluginCapability{
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
}
