package config

// NginxTemplate represent struct for nginx configuration
type NginxTemplate struct {
	ServerName string
	PortPhp    int
	PortNode   int
	RefSlug    string
}
