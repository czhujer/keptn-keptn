package common

import "fmt"

func GetProjectConfigPath(project string) string {
	return fmt.Sprintf("%s/%s", ConfigDir, project)
}

func GetServiceConfigPath(project, service string) string {
	return fmt.Sprintf("%s/%s", GetProjectConfigPath(project), service)
}