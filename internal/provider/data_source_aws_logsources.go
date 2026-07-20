package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/ExaForce/terraform-provider-exaforceio/internal/provider/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &AWSLogSourcesDataSource{}

type AWSLogSourcesDataSource struct {
	client *client.Client
}

func NewAWSLogSourcesDataSource() datasource.DataSource {
	return &AWSLogSourcesDataSource{}
}

type AWSLogSourcesDataSourceModel struct {
	LogSources types.List `tfsdk:"log_sources"`
}

var logSourceSpecAttrTypes = map[string]attr.Type{
	"log_source_type": types.StringType,
	"cluster_name":    types.StringType,
	"eks_arn":         types.StringType,
}

var logSourceAttrTypes = map[string]attr.Type{
	"name": types.StringType,
	"spec": types.ObjectType{AttrTypes: logSourceSpecAttrTypes},
}

func (d *AWSLogSourcesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_logsources"
}

func (d *AWSLogSourcesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all AWS log sources registered in ExaForce CloudScout for the configured project.",
		Attributes: map[string]schema.Attribute{
			"log_sources": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Registered log sources.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true, Description: "Log source name."},
						"spec": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "Log source specification.",
							Attributes: map[string]schema.Attribute{
								"log_source_type": schema.StringAttribute{Computed: true},
								"cluster_name":    schema.StringAttribute{Computed: true},
								"eks_arn":         schema.StringAttribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

func (d *AWSLogSourcesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *AWSLogSourcesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	sources, err := d.client.ListLogSources(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list AWS log sources", err.Error())
		return
	}

	items := make([]attr.Value, 0, len(sources))
	for _, s := range sources {
		// Derive cluster_name from the EKS ARN suffix (cluster/<name>).
		// Non-EKS log sources will have an empty eks_arn, so cluster_name stays null.
		clusterName := types.StringNull()
		if s.Spec.EksArn != "" {
			parts := strings.Split(s.Spec.EksArn, ":")
			if len(parts) >= 6 {
				if derived := strings.TrimPrefix(parts[5], "cluster/"); derived != "" {
					clusterName = types.StringValue(derived)
				}
			}
		}

		specObj, diags := types.ObjectValue(logSourceSpecAttrTypes, map[string]attr.Value{
			"log_source_type": types.StringValue(s.Spec.LogSourceType),
			"cluster_name":    clusterName,
			"eks_arn":         types.StringValue(s.Spec.EksArn),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		obj, diags := types.ObjectValue(logSourceAttrTypes, map[string]attr.Value{
			"name": types.StringValue(s.Name()),
			"spec": specObj,
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		items = append(items, obj)
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: logSourceAttrTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &AWSLogSourcesDataSourceModel{LogSources: list})...)
}
