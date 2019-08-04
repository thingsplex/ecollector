package utils

import (
	"io/ioutil"
	"strings"
)

func match(route []string, topic []string) bool {
	if len(route) == 0 {
		if len(topic) == 0 {
			return true
		}
		return false
	}

	if len(topic) == 0 {
		if route[0] == "#" {
			return true
		}
		return false
	}

	if route[0] == "#" {
		return true
	}

	if (route[0] == "+") || (route[0] == topic[0]) {
		return match(route[1:], topic[1:])
	}

	return false
}

func RouteIncludesTopic(route, topic string) bool {
	return match(strings.Split(route, "/"), strings.Split(topic, "/"))
}


func GetFhSiteId(path string) string {
	if path == "" {
		path = "/var/www/unicomplex/project/app/data/site.data"
	}
	siteIdb, err := ioutil.ReadFile(path)

	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(siteIdb))
}

func GetNodeAndEndpoint(serviceAddr string) (node,endpoint string) {
	parts := strings.Split(serviceAddr,"_")
	if len(parts)==1 {
		return serviceAddr,""
	}else {
		return parts[0],parts[1]
	}
}