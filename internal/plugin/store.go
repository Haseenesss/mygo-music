package plugin

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Topic is the topic on GitHub that a repository with a plugin takes, for the
// app to list it among those to install.
const Topic = "mygo-music-plugin"

// Listing is a repository with a plugin, as GitHub lists it.
type Listing struct {
	Repo        string `json:"repo"` // owner/repo
	Description string `json:"description,omitempty"`
	Stars       int    `json:"stars,omitempty"`
	// Official says it's MyGo Music's own: its author's.
	Official bool `json:"official,omitempty"`
}

// officialOwner is who makes MyGo Music, whose plugins are its own.
const officialOwner = "egoist"

// Official reports whether a repository's plugin is MyGo Music's own, by who
// owns it; the others are the community's, which nobody reviewed.
func Official(repo string) bool {
	owner, _, _ := strings.Cut(repo, "/")
	return strings.EqualFold(owner, officialOwner)
}

// repoFound is a repository as GitHub's search answers it.
type repoFound struct {
	Name        string   `json:"full_name"`
	Description string   `json:"description"`
	Stars       int      `json:"stargazers_count"`
	Archived    bool     `json:"archived"`
	Topics      []string `json:"topics"`
}

// searchRepos asks GitHub's search for repositories, a page of 100.
func searchRepos(query string, page int) (total int, items []repoFound, err error) {
	var found struct {
		Total int         `json:"total_count"`
		Items []repoFound `json:"items"`
	}
	path := fmt.Sprintf("/search/repositories?q=%s&sort=stars&order=desc&per_page=100&page=%d", query, page)
	status, err := githubGet(path, &found)
	if err != nil {
		return 0, nil, err
	}
	if status != http.StatusOK {
		return 0, nil, fmt.Errorf("GitHub answered %d", status)
	}
	return found.Total, found.Items, nil
}

// listedPages is how many pages of 100 Listed asks GitHub for at most; its
// search counts against a tighter limit than the rest of its API.
const listedPages = 3

// Listed lists the plugins on GitHub: MyGo Music's own first, then the public
// repositories with the topic that aren't archived, the most starred first.
// Its author's own are found by their names too, as one may be a fork, which
// GitHub's search leaves out, or lack the topic.
func Listed() ([]Listing, error) {
	listed, err := officialListed()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, l := range listed {
		seen[strings.ToLower(l.Repo)] = true
	}
	for page := 1; page <= listedPages; page++ {
		total, items, err := searchRepos("topic:"+Topic+"+archived:false", page)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			// A page that moved as stars changed may list one again.
			key := strings.ToLower(it.Name)
			if it.Archived || seen[key] || !repoName.MatchString(it.Name) {
				continue
			}
			seen[key] = true
			listed = append(listed, Listing{Repo: it.Name, Description: strings.TrimSpace(it.Description), Stars: it.Stars, Official: Official(it.Name)})
		}
		if len(items) < 100 || page*100 >= total {
			break
		}
	}
	return listed, nil
}

// officialLead is how the names of MyGo Music's own plugins start.
const officialLead = "mygo-music-"

// officialListed are MyGo Music's own plugins: its author's repositories named
// for it, forks among them, those without the topic only if they have a
// plugin.json.
func officialListed() ([]Listing, error) {
	_, items, err := searchRepos("user:"+officialOwner+"+fork:true+mygo-music+in:name+archived:false", 1)
	if err != nil {
		return nil, err
	}
	var listed []Listing
	for _, it := range items {
		_, name, _ := strings.Cut(it.Name, "/")
		if it.Archived || !Official(it.Name) || !repoName.MatchString(it.Name) || !strings.HasPrefix(strings.ToLower(name), officialLead) {
			continue
		}
		if !slices.Contains(it.Topics, Topic) {
			var file struct{}
			status, err := githubGet("/repos/"+it.Name+"/contents/plugin.json", &file)
			if err != nil {
				return nil, err
			}
			if status != http.StatusOK {
				continue
			}
		}
		listed = append(listed, Listing{Repo: it.Name, Description: strings.TrimSpace(it.Description), Stars: it.Stars, Official: true})
	}
	return listed, nil
}
