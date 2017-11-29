package predefined

import (
	"sort"
	"strconv"

	"git.arilot.com/kuberstack/kuberstack-installer/predefined/gen"
	yaml "gopkg.in/yaml.v2"
)

// Products is a variable holding the cluster types defined
var Products []struct {
	ID          int      `yaml:"id"`
	Avatar      string   `yaml:"avatar"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
}

// Tags is a list of all the tags defined with software products
var Tags []string

var id2Name map[string]string

func init() {
	err := yaml.Unmarshal(gen.MustAsset("products.yml"), &Products)
	if err != nil {
		panic(err)
	}

	tagsMap := make(map[string]int, len(Products))

	for _, p := range Products {
		for _, t := range p.Tags {
			tagsMap[t]++
		}
	}

	Tags = make([]string, 0, len(tagsMap))
	for t := range tagsMap {
		Tags = append(Tags, t)
	}

	sort.Strings(Tags)

	id2Name = make(map[string]string, len(Products))

	for _, p := range Products {
		id2Name[strconv.Itoa(p.ID)] = p.Name
	}
}

// GetProductNameByID returns a product name by a string version of product ID
func GetProductNameByID(ID string) string {
	return id2Name[ID]
}
