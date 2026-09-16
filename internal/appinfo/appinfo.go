package appinfo

import "encoding/json"

// AppInfo contains immutable metadata displayed by the application.
type AppInfo struct {
	CompanyName    string `json:"companyName"`
	ProductName    string `json:"productName"`
	ProductVersion string `json:"productVersion"`
	License        string `json:"license"`
	DocsLink       string `json:"docsLink"`
	GHLink         string `json:"ghLink"`
	OrgLink        string `json:"orgLink"`
}

// FromWailsConfig reads application metadata from wails.json.
func FromWailsConfig(wailsJSON []byte) AppInfo {
	var config struct {
		Info AppInfo `json:"info"`
	}
	_ = json.Unmarshal(wailsJSON, &config)

	info := config.Info
	if info.CompanyName == "" {
		info.CompanyName = "raml-dev"
	}
	if info.ProductName == "" {
		info.ProductName = "marktyp"
	}
	if info.ProductVersion == "" {
		info.ProductVersion = "dev"
	}
	if info.License == "" {
		info.License = "GNU AGPL-3.0-only license"
	}
	if info.GHLink == "" {
		info.GHLink = defaultRepositoryURL
	}
	if info.OrgLink == "" {
		info.OrgLink = "https://github.com/raml-dev"
	}

	return info
}
