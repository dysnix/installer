package software

import (
	"strconv"
	"strings"

	"git.arilot.com/kuberstack/kuberstack-installer/db"
	"git.arilot.com/kuberstack/kuberstack-installer/predefined"
	"git.arilot.com/kuberstack/kuberstack-installer/protocol/gen/models"
	"git.arilot.com/kuberstack/kuberstack-installer/savedstate"
)

// GetProducts returns a copy of predefined products array
func GetProducts(search string, tags []string) []*models.GetSoftwareProductsOKBodyItems {
	res := make([]*models.GetSoftwareProductsOKBodyItems, 0, len(predefined.Products))

	for _, p := range predefined.Products {
		if search == "" || strings.Contains(p.Name, search) {
			if len(tags) == 0 || strSlicesCrossed(p.Tags, tags) {
				res = append(
					res,
					&models.GetSoftwareProductsOKBodyItems{
						ID:          strconv.Itoa(p.ID),
						Avatar:      p.Avatar,
						Name:        p.Name,
						Description: p.Description,
						Tags:        p.Tags,
					},
				)
			}
		}
	}
	return res
}

// GetTags returns a list of all the defined tags
func GetTags() models.StringArray {
	return predefined.Tags
}

// Save saves products config to the DB
func Save(
	conn db.Connect,
	products []string,
	principal savedstate.Principal,
) error {
	principal.Sess.Products = products

	return conn.SaveState(principal.ID, principal.Sess)
}

func strSlicesCrossed(s1 []string, s2 []string) bool {
	for _, v1 := range s1 {
		for _, v2 := range s2 {
			if v1 == v2 {
				return true
			}
		}
	}

	return false
}
