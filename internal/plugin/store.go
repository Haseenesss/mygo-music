package plugin

import (
	"fmt"
	"net/http"
	"strings"
)

// Topic is the topic on GitHub that a repository with a plugin takes, for the
// app to list it among those to install.
const Topic = "mygo-music-plugin"

// Listing is a repository with the topic, as GitHub lists it.
type Listing struct {
	Repo        string `json:"repo"` // owner/repo
	Description string `json:"description,omitempty"`
	Stars       int    `json:"stars,omitempty"`
}

// officialOwner is who makes MyGo Music, whose plugins are its own.
const officialOwner = "egoist"

// Official reports whether a repository's plugin is MyGo Music's own, by who
// owns it; the others are the community's, which nobody reviewed.
func Official(repo string) bool {
	owner, _, _ := strings.Cut(repo, "/")
	return strings.EqualFold(owner, officialOwner)
}

// listedPages is how many pages of 100 Listed asks GitHub for at most; its
// search counts against a tighter limit than the rest of its API.
const listedPages = 3

// Listed lists the public repositories with the topic that aren't archived, the
// most starred first.
func Listed() ([]Listing, error) {
	var listed []Listing
	seen := map[string]bool{}
	for page := 1; page <= listedPages; page++ {
		var found struct {
			Total int `json:"total_count"`
			Items []struct {
				Name        string `json:"full_name"`
				Description string `json:"description"`
				Stars       int    `json:"stargazers_count"`
				Archived    bool   `json:"archived"`
			} `json:"items"`
		}
		path := fmt.Sprintf("/search/repositories?q=topic:%s+archived:false&sort=stars&order=desc&per_page=100&page=%d", Topic, page)
		status, err := githubGet(path, &found)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("GitHub answered %d", status)
		}
		for _, it := range found.Items {
			// A page that moved as stars changed may list one again.
			key := strings.ToLower(it.Name)
			if it.Archived || seen[key] || !repoName.MatchString(it.Name) {
				continue
			}
			seen[key] = true
			listed = append(listed, Listing{Repo: it.Name, Description: strings.TrimSpace(it.Description), Stars: it.Stars})
		}
		if len(found.Items) < 100 || page*100 >= found.Total {
			break
		}
	}
	return listed, nil
}
