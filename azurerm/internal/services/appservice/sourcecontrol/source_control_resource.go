package sourcecontrol

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-12-01/web"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/sdk"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/appservice/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/appservice/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/pluginsdk"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

type AppServiceSourceControlResource struct{}

type AppServiceSourceControlModel struct {
	AppID                     string                      `tfschema:"app_id"`
	SCMType                   string                      `tfschema:"scm_type"`
	RepoURL                   string                      `tfschema:"repo_url"`
	Branch                    string                      `tfschema:"branch"`
	ManualIntegration         bool                        `tfschema:"manual_integration"`
	UseMercurial              bool                        `tfschema:"use_mercurial"`
	RollbackEnabled           bool                        `tfschema:"rollback_enabled"`
	UsesGithubAction          bool                        `tfschema:"uses_github_action"`
	GithubActionConfiguration []GithubActionConfiguration `tfschema:"github_action_configuration"`
}

var _ sdk.Resource = AppServiceSourceControlResource{}
var _ sdk.ResourceWithUpdate = AppServiceSourceControlResource{}

func (r AppServiceSourceControlResource) Arguments() map[string]*pluginsdk.Schema {
	return map[string]*pluginsdk.Schema{
		"app_id": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ValidateFunc: validate.WebAppID,
		},

		"repo_url": {
			Type:     pluginsdk.TypeString,
			Required: true,
			// TODO - Validation?
		},

		"scm_type": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			Computed: true,
			ValidateFunc: validation.StringInSlice([]string{
				string(web.ScmTypeBitbucketGit),
				string(web.ScmTypeBitbucketHg),
				string(web.ScmTypeCodePlexGit),
				string(web.ScmTypeCodePlexHg),
				string(web.ScmTypeDropbox),
				string(web.ScmTypeExternalGit),
				string(web.ScmTypeExternalHg),
				string(web.ScmTypeGitHub),
				string(web.ScmTypeLocalGit),
				string(web.ScmTypeNone), // Default when not configured, remove this?
				string(web.ScmTypeOneDrive),
				string(web.ScmTypeTfs),
				string(web.ScmTypeVSO),
				string(web.ScmTypeVSTSRM), // Should this be supported since it's controlled from ADO?
			}, false),
		},

		"branch": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},

		"manual_integration": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Default:  true,
		},

		"uses_github_action": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Default:  false,
		},

		"github_action_configuration": githubActionConfigSchema(),

		"use_mercurial": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Default:  false,
		},

		"rollback_enabled": {
			Type:     pluginsdk.TypeBool,
			Optional: true,
			Default:  false,
		},
	}
}

func (r AppServiceSourceControlResource) Attributes() map[string]*pluginsdk.Schema {
	return nil
}

func (r AppServiceSourceControlResource) ModelObject() interface{} {
	return AppServiceSourceControlModel{}
}

func (r AppServiceSourceControlResource) ResourceType() string {
	return "azurerm_app_service_source_control" // TODO - Does this name fit the new convention?
}

func (r AppServiceSourceControlResource) Create() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			var appSourceControl AppServiceSourceControlModel

			if err := metadata.Decode(&appSourceControl); err != nil {
				return err
			}

			client := metadata.Client.AppService.WebAppsClient

			id, err := parse.WebAppID(appSourceControl.AppID)
			if err != nil {
				return err
			}
			existing, err := client.GetConfiguration(ctx, id.ResourceGroup, id.SiteName)
			if err != nil || existing.SiteConfig == nil {
				return fmt.Errorf("checking for existing Source Control configuration on %s: %+v", id, err)
			}
			if existing.SiteConfig.ScmType != web.ScmTypeNone {
				return metadata.ResourceRequiresImport(r.ResourceType(), id)
			}

			sourceControl := web.SiteSourceControl{
				SiteSourceControlProperties: &web.SiteSourceControlProperties{
					RepoURL:                   utils.String(appSourceControl.RepoURL),
					Branch:                    utils.String(appSourceControl.Branch),
					IsManualIntegration:       utils.Bool(appSourceControl.ManualIntegration),
					IsGitHubAction:            utils.Bool(appSourceControl.UsesGithubAction),
					DeploymentRollbackEnabled: utils.Bool(appSourceControl.RollbackEnabled),
					IsMercurial:               utils.Bool(appSourceControl.UseMercurial),
				},
			}

			if len(appSourceControl.GithubActionConfiguration) != 0 {
				sourceControl.SiteSourceControlProperties.GitHubActionConfiguration = expandGithubActionConfig(appSourceControl.GithubActionConfiguration)
			}

			createFuture, err := client.CreateOrUpdateSourceControl(ctx, id.ResourceGroup, id.SiteName, sourceControl)
			if err != nil {
				return fmt.Errorf("creating Source Control configuration for %s: %v", id, err)
			}

			if err := createFuture.WaitForCompletionRef(ctx, client.Client); err != nil {
				return fmt.Errorf("waiting for Source Control Configuration for %s: %+v", id, err)
			}

			metadata.SetID(id)
			return nil
		},
	}
}

func (r AppServiceSourceControlResource) Read() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 5 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.AppService.WebAppsClient
			id, err := parse.WebAppID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			appSourceControl, err := client.GetSourceControl(ctx, id.ResourceGroup, id.SiteName)
			if err != nil || appSourceControl.SiteSourceControlProperties == nil {
				if utils.ResponseWasNotFound(appSourceControl.Response) {
					return metadata.MarkAsGone(id)
				}
				return fmt.Errorf("reading Source Control for %s: %v", id, err)
			}

			siteConfig, err := client.GetConfiguration(ctx, id.ResourceGroup, id.SiteName)
			if err != nil {
				return fmt.Errorf("reading App for Source Control %s: %v", id, err)
			}

			props := *appSourceControl.SiteSourceControlProperties

			state := AppServiceSourceControlModel{
				AppID:                     id.ID(),
				SCMType:                   string(siteConfig.ScmType),
				RepoURL:                   utils.NormalizeNilableString(props.RepoURL),
				Branch:                    utils.NormalizeNilableString(props.Branch),
				ManualIntegration:         *props.IsManualIntegration,
				UseMercurial:              *props.IsMercurial,
				RollbackEnabled:           *props.DeploymentRollbackEnabled,
				UsesGithubAction:          *props.IsGitHubAction,
				GithubActionConfiguration: flattenGitHubActionConfiguration(props.GitHubActionConfiguration),
			}

			return metadata.Encode(&state)
		},
	}
}

func (r AppServiceSourceControlResource) Delete() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			client := metadata.Client.AppService.WebAppsClient
			id, err := parse.WebAppID(metadata.ResourceData.Id())
			if err != nil {
				return err
			}

			if _, err := client.DeleteSourceControl(ctx, id.ResourceGroup, id.SiteName, ""); err != nil {
				return fmt.Errorf("deleting Source Control for %s: %v", id, err)
			}

			return nil
		},
	}
}

func (r AppServiceSourceControlResource) IDValidationFunc() pluginsdk.SchemaValidateFunc {
	// This is r meta resource with r 1:1 relationship with the service it's pointed at so we use the same ID
	return validate.WebAppID
}

func (r AppServiceSourceControlResource) Update() sdk.ResourceFunc {
	return sdk.ResourceFunc{
		Timeout: 30 * time.Minute,
		Func: func(ctx context.Context, metadata sdk.ResourceMetaData) error {
			return nil
		},
	}
}
