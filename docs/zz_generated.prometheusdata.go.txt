// CustomizationData is the data available to custom scraping configs and rules,
// containing everything required to scrape resources. This is a public interface
// and changes to this struct could break existing custom scrape/rule configs, so
// care must be taken when changing this.
type CustomizationData struct {
	Cluster                  *kubermaticv1.Cluster
	APIServerHost            string
	EtcdTLS                  TLSConfig
	ApiserverTLS             TLSConfig
	ScrapingAnnotationPrefix string
}

type TLSConfig struct {
	CAFile   string `yaml:"ca_file"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type TLSConfig struct {
	CAFile   string `yaml:"ca_file"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}
