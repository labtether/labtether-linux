package webservice

// currentAssetID follows the Hub-approved identity after pending enrollment.
// Other Transport implementations retain the ID supplied at construction.
func (wsc *WebServiceCollector) currentAssetID() string {
	if source, ok := wsc.transport.(interface{ AssetID() string }); ok {
		if assetID := source.AssetID(); assetID != "" {
			return assetID
		}
	}
	return wsc.assetID
}
