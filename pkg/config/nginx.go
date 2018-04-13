package config

// NginxTemplate represent struct for nginx configuration
type NginxTemplate struct {
	ServerName   string
	PortPhp      int
	PortNode     int
	RefSlug      string
	TemplatePath string
}

// Write create nginx configuration files for virtual hosts
func (t *NginxTemplate) Write(path string) error {
	conf := ParseTemplate(t.TemplatePath, t)
	return WriteStringToFile(path, conf)
}
