package cluster

import (
	"fmt"

	"git.arilot.com/kuberstack/kuberstack-installer/predefined"
	"git.arilot.com/kuberstack/kuberstack-installer/protocol/gen/models"
)

// PriceFormat is a printf-compatible format for a string price representation
const PriceFormat string = "%.2f"

// GetTypes returns a copy of cluster types list
func GetTypes() models.GetClusterTypesOKBodyTypes {
	res := make(models.GetClusterTypesOKBodyTypes, 0, len(predefined.Types))

	for _, t := range predefined.Types {
		res = append(
			res,
			&models.ConfigType{
				ID:          int64(t.ID),
				Name:        t.Name,
				ShortName:   t.ShortName,
				Description: t.Description,
				Price:       fmt.Sprintf(PriceFormat, t.Price),
			},
		)
	}
	return res
}
