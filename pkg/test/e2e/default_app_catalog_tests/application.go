package default_app_catalog_applications_tests

type ApplicationInterface interface {
	GetApplication(version string) ([]byte, error)
	FetchData() (name, namespace, key string, names []string)
}
