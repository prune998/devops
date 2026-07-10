package lister

import "fmt"

// Repository is a single repository result, shaped for JSON output.
type Repository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
}

// SearchQuery builds the GitHub search query string that matches all
// repositories in org carrying the given topic.
func SearchQuery(org, topic string) string {
	return fmt.Sprintf("org:%s topic:%s", org, topic)
}
