package client

const (
	CatalogSpecType             = "catalogSpec"
	CatalogSpecFieldBranch      = "branch"
	CatalogSpecFieldCatalogKind = "catalogKind"
	CatalogSpecFieldDescription = "description"
	CatalogSpecFieldPassword    = "password"
	CatalogSpecFieldURL         = "url"
	CatalogSpecFieldUsername    = "username"
)

type CatalogSpec struct {
	Branch      string `json:"branch,omitempty" yaml:"branch,omitempty"`
	CatalogKind string `json:"catalogKind,omitempty" yaml:"catalogKind,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Password    string `json:"password,omitempty" yaml:"password,omitempty"`
	URL         string `json:"url,omitempty" yaml:"url,omitempty"`
	Username    string `json:"username,omitempty" yaml:"username,omitempty"`
}
