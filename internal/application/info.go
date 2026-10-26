package application

// AppInfo is the product metadata presented by both command-line interfaces.
type AppInfo struct {
	Name    string
	Version string
	Tagline string
}

// Info returns the metadata for this build of ProvisionHub.
func Info() AppInfo {
	return AppInfo{
		Name:    "ProvisionHub",
		Version: "0.1.0-dev",
		Tagline: "Developer Provisioning Toolkit",
	}
}
