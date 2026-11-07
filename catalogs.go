package main

// The TUI serves several catalogs ("modes"): setup, footprinting, the web
// pentest catalog, the Active Directory catalog, the Linux local-privilege-
// escalation catalog, and so on. H/L in the tool list switches
// between them.
type catalogMode struct {
	Key   string // stable id
	Label string // shown on the catalog switcher
	tools func() []Tool
}

// catalogModes returns the available catalogs. It is a function (not a package
// var) so the catalogs are read after payloads.go's init() has appended to
// them.
func catalogModes() []catalogMode {
	return []catalogMode{
		{Key: "setup", Label: "Setup", tools: func() []Tool { return setupCatalog }},
		{Key: "footprinting", Label: "Footprinting", tools: func() []Tool { return footprintingCatalog }},
		{Key: "web", Label: "Web", tools: func() []Tool { return catalog }},
		{Key: "ad", Label: "AD", tools: func() []Tool { return adCatalog }},
		{Key: "linux", Label: "Linux", tools: func() []Tool { return linuxCatalog }},
		{Key: "revshell", Label: "Reverse Shells", tools: func() []Tool { return revshellCatalog }},
		{Key: "filetransfer", Label: "File Transfers", tools: func() []Tool { return fileTransferCatalog }},
	}
}
