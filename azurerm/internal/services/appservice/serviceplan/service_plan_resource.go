package serviceplan

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-12-01/web"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/validation"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/appservice/parse"

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

type OSType string

const (
	OSTypeLinux            OSType = "Linux"
	OSTypeWindows          OSType = "Windows"
	OSTypeWindowsContainer OSType = "WindowsContainer"
)

type AppServicePlanModel struct {
	Name                      string            `tfschema:"name"`
	ResourceGroup             string            `tfschema:"resource_group_name"`
	Location                  string            `tfschema:"location"`
	Kind                      string            `tfschema:"kind"` // Computed Only
	OSType                    OSType            `tfschema:"os_type"`
	Sku                       string            `tfschema:"sku_name"`
	AppServiceEnvironmentId   string            `tfschema:"app_service_environment_id"`
	PerSiteScaling            bool              `tfschema:"per_site_scaling"`
	Reserved                  bool              `tfschema:"reserved"` // Computed Only?
	NumberOfWorkers           int               `tfschema:"number_of_workers"`
	MaximumElasticWorkerCount int               `tfschema:"maximum_elastic_worker_count"`
	Tags                      map[string]string `tfschema:"tags"`
	// TODO properties
	// KubernetesID string `tfschema:"kubernetes_id"` // AKS Cluster resource ID?
}

func (r AppServicePlanResource) Arguments() map[string]*schema.Schema {
	return map[string]*pluginsdk.Schema{
		"name": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ValidateFunc: validate.ServicePlanName,
		},

		"resource_group_name": azure.SchemaResourceGroupName(),

		"location": location.Schema(),

		"sku_name": {
			Type:     pluginsdk.TypeString,
			Required: true,
			// TODO - Validation
			// Note - need to look at Isolated as separate property via ExactlyOneOf?
		},

		"os_type": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			Default:  string(OSTypeWindows),
			ValidateFunc: validation.StringInSlice([]string{
				string(OSTypeLinux),
				string(OSTypeWindows),
				string(OSTypeWindowsContainer),
			}, false),
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

		"number_of_workers": {
			Type:         pluginsdk.TypeInt,
			Optional:     true,
			Computed:     true,
			ValidateFunc: nil, // TODO - IntAtLeast?
		},

		"maximum_elastic_worker_count": {
			Type:         pluginsdk.TypeInt,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.IntAtLeast(0),
		},

		"tags": tags.Schema(),
	}
}

func (r AppServicePlanResource) Attributes() map[string]*schema.Schema {
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

func (r AppServicePlanResource) ModelObject() interface{} {
	return AppServicePlanModel{}
}

func (r AppServicePlanResource) ResourceType() string {
	return "azurerm_service_plan"
}

func (r AppServicePlanResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			var servicePlan AppServicePlanModel
			if err := metadata.Decode(&servicePlan); err != nil {
				return err
			}

			client := metadata.Client.AppService.ServicePlanClient
			subscriptionId := metadata.Client.Account.SubscriptionId

			id := parse.NewServicePlanID(subscriptionId, servicePlan.ResourceGroup, servicePlan.Name)

			existing, err := client.Get(ctx, id.ResourceGroup, id.ServerfarmName)
			if err != nil && !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for existing Service Plan %s: %v", id, err)
			}
			if !utils.ResponseWasNotFound(existing.Response) {
				return metadata.ResourceRequiresImport(r.ResourceType(), id)
			}

			appServicePlan := web.AppServicePlan{
				AppServicePlanProperties: &web.AppServicePlanProperties{
					PerSiteScaling: utils.Bool(servicePlan.PerSiteScaling),
					Reserved:       utils.Bool(servicePlan.OSType == OSTypeLinux),
					HyperV:         utils.Bool(servicePlan.OSType == OSTypeWindowsContainer),
				},
				Sku: &web.SkuDescription{
					Name: utils.String(servicePlan.Sku),
				},
				Location: utils.String(location.Normalize(servicePlan.Location)),
				Tags:     tags.FromTypedObject(servicePlan.Tags),
			}

			if servicePlan.MaximumElasticWorkerCount > 0 {
				appServicePlan.AppServicePlanProperties.MaximumElasticWorkerCount = utils.Int32(int32(servicePlan.MaximumElasticWorkerCount))
			}

			if servicePlan.NumberOfWorkers != 0 {
				appServicePlan.Sku.Capacity = utils.Int32(int32(servicePlan.NumberOfWorkers))
			}

			future, err := client.CreateOrUpdate(ctx, id.ResourceGroup, id.ServerfarmName, appServicePlan)
			if err != nil {
				return fmt.Errorf("creating %s: %v", id, err)
			}

			if err := future.WaitForCompletionRef(ctx, client.Client); err != nil {
				return fmt.Errorf("waitng for creation of %s: %v", id, err)
			}

			metadata.SetID(id)

			return nil
		},
	}
}

func (r AppServicePlanResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.AppService.ServicePlanClient
			id, err := parse.ServicePlanID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			servicePlan, err := client.Get(ctx, id.ResourceGroup, id.ServerfarmName)
			if err != nil {
				if utils.ResponseWasNotFound(servicePlan.Response) {
					return metadata.MarkAsGone(id)
				}
				return fmt.Errorf("reading %s: %+v", id, err)
			}

			state := AppServicePlanModel{
				Name:          id.ServerfarmName,
				ResourceGroup: id.ResourceGroup,
				Location:      location.NormalizeNilable(servicePlan.Location),
				Kind:          utils.NormalizeNilableString(servicePlan.Kind),
			}

			// sku read
			if sku := servicePlan.Sku; sku != nil {
				if sku.Name != nil {
					state.Sku = *sku.Name
					if sku.Capacity != nil {
						state.NumberOfWorkers = int(*sku.Capacity)
					}
				}
			}

			// props read
			if props := servicePlan.AppServicePlanProperties; props != nil {
				state.OSType = OSTypeWindows
				if props.HyperV != nil && *props.HyperV {
					state.OSType = OSTypeWindowsContainer
				}
				if props.Reserved != nil && *props.Reserved {
					state.OSType = OSTypeLinux
				}

				if props.MaximumElasticWorkerCount != nil {
					state.MaximumElasticWorkerCount = int(*props.MaximumElasticWorkerCount)
				}

				if ase := props.HostingEnvironmentProfile; ase != nil && ase.ID != nil {
					state.AppServiceEnvironmentId = *ase.ID
				}

				state.PerSiteScaling = *props.PerSiteScaling
				state.Reserved = *props.Reserved

				state.MaximumElasticWorkerCount = int(utils.NormaliseNilableInt32(props.MaximumElasticWorkerCount))
			}
			state.Tags = tags.ToTypedObject(servicePlan.Tags)

			return metadata.Encode(&state)
		},
	}
}

func (r AppServicePlanResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			id, err := parse.ServicePlanID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			client := metadata.Client.AppService.ServicePlanClient
			metadata.Logger.Infof("deleting %s", id)

			if _, err := client.Delete(ctx, id.ResourceGroup, id.ServerfarmName); err != nil {
				return fmt.Errorf("deleting %s: %v", id, err)
			}

			return nil
		},
	}
}

func (r AppServicePlanResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	return validate.ServicePlanID
}

func (r AppServicePlanResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 60 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			return nil
		},
	}
}
