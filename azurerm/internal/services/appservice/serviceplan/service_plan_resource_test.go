package serviceplan_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/acceptance/check"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/appservice/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/pluginsdk"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

type ServicePlanResource struct{}

func TestAccServicePlan_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_service_plan", "test")
	r := ServicePlanResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.basic(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccServicePlan_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_service_plan", "test")
	r := ServicePlanResource{}

	data.ResourceTest(t, r, []resource.TestStep{
		{
			Config: r.basic(data),
			Check: resource.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.RequiresImportErrorStep(r.requiresImport),
	})
}

func (r ServicePlanResource) Exists(ctx context.Context, client *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := parse.ServicePlanID(state.ID)
	if err != nil {
		return nil, err
	}

	resp, err := client.AppService.ServicePlanClient.Get(ctx, id.ResourceGroup, id.ServerfarmName)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return utils.Bool(false), nil
		}
		return nil, fmt.Errorf("retreiving %s: %v", id, err)
	}
	if utils.ResponseWasNotFound(resp.Response) {
		return utils.Bool(false), nil
	}
	return utils.Bool(true), nil
}

// Configs

func (r ServicePlanResource) basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-SP-%[1]d"
  location = "%s"
}

resource "azurerm_service_plan" "test" {
  name                = "acctest-SP-%[1]d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location
  sku_name            = "B1"

  tags = {
    environment = "AccTest"
    Foo         = "bar"
  }
}
`, data.RandomInteger, data.Locations.Primary)
}

func (r ServicePlanResource) requiresImport(data acceptance.TestData) string {
	return fmt.Sprintf(`

%s

resource "azurerm_service_plan" "import" {
  name                = azurerm_service_plan.test.name
  resource_group_name = azurerm_service_plan.test.resource_group_name
  location            = azurerm_service_plan.test.location
  sku_name            = azurerm_service_plan.test.sku_name
}
`, r.basic(data))
}
