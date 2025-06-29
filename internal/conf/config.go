package conf

var (
	Conf = new(Config)
)

type Config struct {
	S3 S3Config `yaml:"s3"`
}

type S3Config struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Bucket          string `yaml:"bucket"`
	CDNBaseURL      string `yaml:"cdn_base_url"`
	UseSSL          bool   `yaml:"use_ssl"`
}

func (c *Config) Print() {
}
