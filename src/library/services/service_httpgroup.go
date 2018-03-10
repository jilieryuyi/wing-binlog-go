package services

import "library/app"

func newHttpGroup(groupConfig app.HttpNodeConfig) *httpGroup {
	group := &httpGroup{
		name: groupConfig.Name,
		filter: groupConfig.Filter,
		nodes: nil,
	}
	for _, url := range groupConfig.Nodes {
		group.nodes = append(group.nodes, newHttpNode(url))
	}
	return group
}
