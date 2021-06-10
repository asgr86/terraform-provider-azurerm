package serviceplan

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/location"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/sdk"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/appservice/validate"
	webValidate "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/web/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tags"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/pluginsdk"
)

type AppServicePlanResource struct{}

var _ sdk.Resource = AppServicePlanResource{}
var _ sdk.ResourceWithUpdate = AppServicePlanResource{}

type AppServicePlanModel struct {
	Name                      string `tfschema:"name"`
	ResourceGroup             string `tfschema:"resource_group_name"`
	Location                  string `tfschema:"location"`
	Kind                      string `tfschema:"kind"` // Computed Only
	Sku                       string `tfschema:"sku_name"`
	AppServiceEnvironmentId   string `tfschema:"app_service_environment_id"`
	PerSiteScaling            bool   `tfschema:"per_site_scaling"`
	Reserved                  bool   `tfschema:"reserved"` // Computed Only?
	MaximumElasticWorkerCount int    `tfschema:"maximum_elastic_worker_count"`
	// Xenon bool - Still valid?
	// HyperV bool - Still valid?
	Tags map[string]string `tfschema:"tags"`
}

func (a AppServicePlanResource) Arguments() map[string]*schema.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ValidateFunc: validate.ServicePlanID,
		},

		"resource_group_name": azure.SchemaResourceGroupName(),

		"location": location.Schema(),

		"sku_name": {
			Type:     pluginsdk.TypeString,
			Required: true,
			// TODO - Validation
			// Note - need to look at Isolated as separate property via ExactlyOneOf?
		},

		"app_service_environment_id": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ValidateFunc: webValidate.AppServiceEnvironmentID, // TODO - Bring over to this service
		},

		"per_site_scaling": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Default:  false,
		},

		"maximum_elastic_worker_count": {
			Type:     pluginsdk.TypeInt,
			Optional: true,
			Computed: true,
		},

		"tags": tags.Schema(),
	}
}

func (a AppServicePlanResource) Attributes() map[string]*schema.Schema {
	return map[string]*pluginsdk.Schema{
		"kind": {
			Type:     pluginsdk.TypeString,
			Computed: true,
		},

		"reserved": {
			Type:     pluginsdk.TypeBool,
			Computed: true,
		},
	}
}

func (a AppServicePlanResource) ModelObject() interface{} {
	return AppServicePlanModel{}
}

func (a AppServicePlanResource) ResourceType() string {
	return "azurerm_service_plan"
}

func (a AppServicePlanResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			return nil
		},
	}
}

func (a AppServicePlanResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			return nil
		},
	}
}

func (a AppServicePlanResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			return nil
		},
	}
}

func (a AppServicePlanResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return validate.ServicePlanID
}

func (a AppServicePlanResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			return nil
		},
	}
}
