package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Commit struct {
	Sha string `json:"sha,omitempty"`
	Url string `json:"url,omitempty"`
}

type ReleaseNotes struct {
	ReleaseName string   `json:"release_name,omitempty"`
	Version     string   `json:"version,omitempty"`
	Fixes       []string `json:"fixes,omitempty"`
	Features    []string `json:"features,omitempty"`
	Maintenance []string `json:"maintenance,omitempty"`
	Changes     []string `json:"changes,omitempty"`
}

type Release struct {
	Name         string        `json:"name,omitempty"`
	ZipballUrl   string        `json:"zipball_url,omitempty"`
	TarballUrl   string        `json:"tarball_url,omitempty"`
	Commit       Commit        `json:"commit,omitempty"`
	ReleaseNotes *ReleaseNotes `json:"release_notes,omitempty"`
}

var releases []Release

func wget(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (r *Release) dump() {
	fmt.Println("Name:", r.Name)
	fmt.Println("ZipballUrl:", r.ZipballUrl)
	fmt.Println("TarballUrl:", r.TarballUrl)
	fmt.Println("Commit:")
	fmt.Println("\tSHA:", r.Commit.Sha)
	fmt.Println("\tURL:", r.Commit.Url)
	fmt.Println("Release Notes:")
	fmt.Println("\tVersion:", r.ReleaseNotes.Version)
	fmt.Println("\tFixes:")
	for _, f := range r.ReleaseNotes.Fixes {
		fmt.Println("\t\t *", f)
	}
	fmt.Println("\tFeatures:")
	for _, f := range r.ReleaseNotes.Features {
		fmt.Println("\t\t *", f)
	}
	fmt.Println("\tMaintenance:")
	for _, m := range r.ReleaseNotes.Maintenance {
		fmt.Println("\t\t *", m)
	}
	fmt.Println("\tChanges:")
	for _, c := range r.ReleaseNotes.Changes {
		fmt.Println("\t\t *", c)
	}
	fmt.Println()
}

// TODO: fix these loops with maps
func insertFix(release string, name string, line string) {
	for i, r := range releases {
		if r.Name == name {
			if releases[i].ReleaseNotes == nil {
				releases[i].ReleaseNotes = &ReleaseNotes{}
			}
			releases[i].ReleaseNotes.ReleaseName = release
			releases[i].ReleaseNotes.Version = name
			releases[i].ReleaseNotes.Fixes = append(releases[i].ReleaseNotes.Fixes, line)
		}
	}
}

func insertFeature(release string, name string, line string) {
	for i, r := range releases {
		if r.Name == name {
			if releases[i].ReleaseNotes == nil {
				releases[i].ReleaseNotes = &ReleaseNotes{}
			}
			releases[i].ReleaseNotes.ReleaseName = release
			releases[i].ReleaseNotes.Version = name
			releases[i].ReleaseNotes.Features = append(releases[i].ReleaseNotes.Features, line)
		}
	}
}

func insertMaintenance(release string, name string, line string) {
	for i, r := range releases {
		if r.Name == name {
			if releases[i].ReleaseNotes == nil {
				releases[i].ReleaseNotes = &ReleaseNotes{}
			}
			releases[i].ReleaseNotes.ReleaseName = release
			releases[i].ReleaseNotes.Version = name
			releases[i].ReleaseNotes.Maintenance = append(releases[i].ReleaseNotes.Maintenance, line)
		}
	}
}

func insertChange(release string, name string, line string) {
	for i, r := range releases {
		if r.Name == name {
			if releases[i].ReleaseNotes == nil {
				releases[i].ReleaseNotes = &ReleaseNotes{}
			}
			releases[i].ReleaseNotes.ReleaseName = release
			releases[i].ReleaseNotes.Version = name
			releases[i].ReleaseNotes.Changes = append(releases[i].ReleaseNotes.Changes, line)
		}
	}
}

func parseChangelog(changelog string) {
	validVersion := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`)

	release := ""
	name := ""
	comment := ""
	fixes := false
	features := false
	maintenance := false

	scanner := bufio.NewScanner(strings.NewReader(changelog))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t\n\v\f\r")
		if len(line) == 0 {
			continue
		}
		// buffer comments first to handle multilined comments
		if len(line) >= 1 && line[0] != '#' && line[0] != '*' {
			comment += "\n" + strings.TrimSpace(line)
			continue
		}
		// handle buffered comments
		if comment != "" {
			if fixes {
				insertFix(release, name, comment)
			} else if features {
				insertFeature(release, name, comment)
			} else if maintenance {
				insertMaintenance(release, name, comment)
			} else {
				insertChange(release, name, comment)
			}
		}
		// Releases
		if len(line) >= 3 && line[:3] == "## " {
			release = line[3:]
			// NOTE: doesn't handle edge case "jest 22.0.2 && 22.0.3"
			name = "v" + string(validVersion.Find([]byte(release)))
			comment = ""
			fixes = false
			features = false
			maintenance = false
		}
		if !validVersion.MatchString(release) {
			continue
		}
		// Fixes / Features / Maintenance
		if len(line) >= 4 && line[:4] == "### " {
			if strings.Contains(line, "Fixes") {
				comment = ""
				fixes = true
				features = false
				maintenance = false
			} else if strings.Contains(line, "Features") {
				comment = ""
				fixes = false
				features = true
				maintenance = false
			} else if strings.Contains(line, "Chore & Maintenance") {
				comment = ""
				fixes = false
				features = false
				maintenance = true
			}
		}
		// Comments
		if len(line) >= 2 && line[:2] == "* " {
			comment = line[2:]
		}
	}
}

const changelogURL = "https://raw.githubusercontent.com/facebook/jest/master/CHANGELOG.md"
const tagsURL = "https://api.github.com/repos/facebook/jest/tags"

func main() {
	// Get tags
	tags, err := wget(tagsURL)
	if err != nil {
		log.Fatalln(err)
	}

	// Unmarshal tags into data structure
	err = json.Unmarshal([]byte(tags), &releases)
	if err != nil {
		log.Fatalln(err)
	}

	// Get changelog
	changelog, err := wget(changelogURL)
	if err != nil {
		log.Fatalln(err)
	}

	// Parse changelog
	parseChangelog(changelog)

	// DEBUG: Dump releases
	// for _, r := range releases {
	// 	r.dump()
	// }

	// Marshal tags into bytes
	b, err := json.Marshal(releases)
	if err != nil {
		log.Fatalln(err)
	}

	// Write to file
	err = ioutil.WriteFile("r2c.json", b, 0644)
	if err != nil {
		log.Fatalln(err)
	}
}
