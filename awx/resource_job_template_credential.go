/*
*TBD*

Example Usage

```hcl
resource "awx_job_template_credentials" "baseconfig" {
  job_template_id = awx_job_template.baseconfig.id
  credential_id   = awx_credential_machine.pi_connection.id
}
```

*/
package awx

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	awx "github.com/mrcrilly/goawx/client"
)

func resourceJobTemplateCredentials() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceJobTemplateCredentialsCreate,
		DeleteContext: resourceJobTemplateCredentialsDelete,
		ReadContext:   resourceJobTemplateCredentialsRead,

		Schema: map[string]*schema.Schema{

			"job_template_id": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"credential_id": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
		},

		Importer: &schema.ResourceImporter{
			State: resourceJobTemplateCredentialsStateImport,
		},
	}
}

func resourceJobTemplateCredentialsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*awx.AWX)
	awxService := client.JobTemplateService
	jobTemplateID := d.Get("job_template_id").(int)
	credentialID := d.Get("credential_id").(int)
	_, err := awxService.GetJobTemplateByID(jobTemplateID, make(map[string]string))
	if err != nil {
		return buildDiagNotFoundFail("job template", jobTemplateID, err)
	}

	_, err = awxService.AssociateCredentials(jobTemplateID, map[string]interface{}{
		"id": credentialID,
	}, map[string]string{})

	if err != nil {
		return buildDiagnosticsMessage("Create: JobTemplate not AssociateCredentials", "Fail to add credentials with Id %v, for Template ID %v, got error: %s", credentialID, jobTemplateID, err.Error())
	}

	d.SetId(fmt.Sprintf("%d-%d", jobTemplateID, credentialID))
	return diags
}


func resourceJobTemplateCredentialsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*awx.AWX)
	awxService := client.JobTemplateService
	jobTemplateID := d.Get("job_template_id").(int)
	credentialID := d.Get("credential_id").(int)

	params := make(map[string]string)
	params["id"] = strconv.Itoa(credentialID)

	creds, _, err := awxService.ListCredentials(jobTemplateID, params)
	if err != nil {
		return buildDiagnosticsMessage(
			"Get: Fail to fetch Credential",
			"Fail to find the credential for job template ID %d and credential ID %d got: %s",
			jobTemplateID, credentialID, err.Error(),
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
		d.SetId("")
		d.Set("job_template_id", "")
		d.Set("credential_id", "")
	}

	return diags
}

func resourceJobTemplateCredentialsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*awx.AWX)
	awxService := client.JobTemplateService
	jobTemplateID := d.Get("job_template_id").(int)
	res, err := awxService.GetJobTemplateByID(jobTemplateID, make(map[string]string))
	if err != nil {
		return buildDiagNotFoundFail("job template", jobTemplateID, err)
	}

	_, err = awxService.DisAssociateCredentials(res.ID, map[string]interface{}{
		"id": d.Get("credential_id").(int),
	}, map[string]string{})
	if err != nil {
		return buildDiagDeleteFail("JobTemplate DisAssociateCredentials", fmt.Sprintf("DisAssociateCredentials %v, from JobTemplateID %v got %s ", d.Get("credential_id").(int), d.Get("job_template_id").(int), err.Error()))
	}

	d.SetId("")
	return diags
}

func resourceJobTemplateCredentialsStateImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), "-")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return nil, fmt.Errorf("[Error] Unexpected format of ID (%q), expected <job_template_id>-<credential_id>", d.Id())
	}
	jobTemplateID, diags := convertStrToNumeric("job template credential import", idParts[0])
	if diags.HasError() {
		return nil, fmt.Errorf("[Error] can't parse job template ID")
	}
	credentialID, diags := convertStrToNumeric("job template credential import", idParts[1])
	if diags.HasError() {
		return nil, fmt.Errorf("[Error] can't parse credential ID")
	}

	d.Set("job_template_id", jobTemplateID)
	d.Set("credential_id", credentialID)
	return []*schema.ResourceData{d}, nil
}
