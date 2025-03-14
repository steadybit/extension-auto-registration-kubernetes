package autoregistration

type extensionConfigAO struct {
	UnixSocket      string         `json:"unixSocket,omitempty"` //important even if not used to be able to delete existing registrations
	Url             string         `json:"url,omitempty"`
	Types           []string       `json:"types,omitempty"`
	RestrictedPorts map[int]string `json:"restrictedPorts,omitempty"`
	RestrictedIps   []string       `json:"restrictedIps,omitempty"`
}

type ExtensionAnnotations struct {
	Extensions []ExtensionAnnotation `json:"extensions,omitempty"`
}
type ExtensionAnnotation struct {
	Protocol string `json:"protocol,omitempty"`
	Port     int    `json:"port,omitempty"`
	Path     string `json:"path,omitempty"`
}
