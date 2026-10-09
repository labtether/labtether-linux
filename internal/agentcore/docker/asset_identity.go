package docker

// currentAssetID follows the Hub-approved identity after pending enrollment.
// Other Transport implementations retain the ID supplied at construction.
func (dc *DockerCollector) currentAssetID() string {
	if source, ok := dc.transport.(interface{ AssetID() string }); ok {
		if assetID := source.AssetID(); assetID != "" {
			return assetID
		}
	}
	return dc.assetID
}
