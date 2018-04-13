package config

// BooksConfig JSON nested configuration for Books
type BooksConfig struct {
	Production  BooksConfigNested `json:"production"`
	Development BooksConfigNested `json:"development"`
}

// BooksEnv JSON database settings in nested structure of BooksEnv
type BooksEnv struct {
	BaseName string `json:"BASE_NAME"`
	UserName string `json:"USER_NAME"`
	Password string `json:"PASSWORD"`
	Host     string `json:"HOST"`
}

// BooksConfigNested extend JSON structure of BooksConfig
type BooksConfigNested struct {
	BooksEnv          BooksEnv `json:"BASE_CONFIG"`
	ExternalServerAPI string   `json:"EXTERNAL_SERVER_API"`
}

func (p BooksConfig) Write(path string) {
	WriteJSONToFile(path, p)
}
