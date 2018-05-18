package rbac

import "fmt"

const (
	ownerGroupName = "owners"
)

func generateOwnersGroupName(projectName string) string {
	return fmt.Sprintf("%s-%s", projectName, ownerGroupName)
}
