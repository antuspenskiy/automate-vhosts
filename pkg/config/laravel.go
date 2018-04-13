package config

// LaravelTemplate laravel environment configuration
type LaravelTemplate struct {
	AppURL       string
	DBDatabase   string
	DBUserName   string
	DBPassword   string
	TemplatePath string
}

// Write used to create laravel environment files for virtual hosts
func (t *LaravelTemplate) Write(path string) {
	conf := ParseTemplate(t.TemplatePath, t)
	WriteToFile(path, conf)
}
