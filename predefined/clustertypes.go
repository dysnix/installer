package predefined

import (
	"git.arilot.com/kuberstack/kuberstack-installer/predefined/gen"
	yaml "gopkg.in/yaml.v2"
)

// Types is a variable holding the cluster types defined
var Types []struct {
	ID          int     `yaml:"id"`
	Name        string  `yaml:"name"`
	ShortName   string  `yaml:"shortName"`
	Description string  `yaml:"description"`
	Price       float64 `yaml:"price"`
}

func init() {
	err := yaml.Unmarshal(gen.MustAsset("clustertypes.yml"), &Types)
	if err != nil {
		panic(err)
	}
}
