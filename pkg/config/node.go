package config

// BooksEnv JSON nested configuration for Books
type BooksEnv struct {
	Production  BooksConfigNested `json:"production"`
	Development BooksConfigNested `json:"development"`
}

// BooksConfig JSON database settings in nested structure of BooksEnv
type BooksConfig struct {
	BaseName string `json:"BASE_NAME"`
	UserName string `json:"USER_NAME"`
	Password string `json:"PASSWORD"`
	Host     string `json:"HOST"`
}

// BooksConfigNested extend JSON structure of BooksConfig
type BooksConfigNested struct {
	BooksConfig       BooksConfig `json:"BASE_CONFIG"`
	ExternalServerAPI string      `json:"EXTERNAL_SERVER_API"`
}
