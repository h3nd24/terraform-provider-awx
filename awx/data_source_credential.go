/*
Use this data source to query Credential by ID.

Example Usage

```hcl
*TBD*
```

*/
package awx

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	awx "github.com/mrcrilly/goawx/client"
)

func dataSourceCredentialByID() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCredentialByIDRead,
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Optional:    true,
				Description: "Credential id",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Credential name",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Username from searched id",
			},
			"kind": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Kind from searched id",
			},
		},
	}
}

func dataSourceCredentialByIDRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*awx.AWX)
	params := make(map[string]string)
	if name, okName := d.GetOk("name"); okName {
		params["name"] = name.(string)
	}

	if credId, okCredID := d.GetOk("id"); okCredID {
		params["id"] = strconv.Itoa(credId.(int))
	}

	if len(params) == 0 {
		return buildDiagnosticsMessage(
			"Get: Missing Parameters",
			"Please use one of the selectors (name or id)",
		)
	}
	creds, _, err := client.CredentialsService.ListCredentials(params)
	if err != nil {
		return buildDiagnosticsMessage(
			"Get: Fail to fetch Credential",
			"Fail to find the credential got: %s",
			err.Error(),
		)
	}
	if len(creds) > 1 {
		return buildDiagnosticsMessage(
			"Get: find more than one Element",
			"The Query Returns more than one Credentials, %d",
			len(creds),
		)
	}
	if len(creds) == 0 {
		return buildDiagnosticsMessage(
			"Get: find no Element",
			"The Query Returns no Credentials, %d",
			len(creds),
		)
	}

	cred := creds[0]
	d.Set("id", cred.ID)
	d.Set("name", cred.Name)
	d.Set("username", cred.Inputs["username"])
	d.Set("kind", cred.Kind)
	d.SetId(strconv.Itoa(cred.ID))
	return diags
}
