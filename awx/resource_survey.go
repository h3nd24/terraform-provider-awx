/*
*TBD*

Example Usage

```hcl
data "awx_inventory" "default" {
  name            = "private_services"
  organisation_id = data.awx_organization.default.id
}

resource "awx_job_template" "baseconfig" {
  name           = "baseconfig"
  job_type       = "run"
  inventory_id   = data.awx_inventory.default.id
  project_id     = awx_project.base_service_config.id
  playbook       = "master-configure-system.yml"
  become_enabled = true
}
```

*/
package awx

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	awx "github.com/mrcrilly/goawx/client"
)

func resourceSurvey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSurveyCreate,
		ReadContext:   resourceSurveyRead,
		UpdateContext: resourceSurveyUpdate,
		DeleteContext: resourceSurveyDelete,

		Schema: map[string]*schema.Schema{
			"job_template_id": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"spec": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Resource{
					Schema: map[string]*schema.Schema{
						"question_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"question_description": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"required": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"variable": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: func(v any, p cty.Path) diag.Diagnostics {
								value := v.(string)
								expectedList := [6]string{"text","multiplechoice","multiselect","password","integer","float"}
								var diags diag.Diagnostics
								for _, expected := range expectedList {
									if value == expected {
										return diags
									}
								}
								// not found in expected list, abort
								diag := diag.Diagnostic{
									Severity: diag.Error,
									Summary:  "wrong value",
									Detail:   fmt.Sprintf("%q is not %q", value, expectedList),
								}
								diags = append(diags, diag)
								return diags
							},
						},
						"min": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"max": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1024,
						},
						"default": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
						"choices": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "",
						},
					},
                                },
			},
		},

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		
	}
}

func resourceSurveyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*awx.AWX)
	awxService := client.JobTemplateService
        surveySpecs, err := getSurveySpecs(d)
	if err != nil {
        	log.Printf("%v", err.Error())
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to add surveys",
			Detail:   fmt.Sprintf("JobTemplate with id %d, failed to add survey: %s", d.Get("job_template_id").(int), err.Error()),
		})
		return diags
	}

	survey := awx.Survey{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Spec:        surveySpecs,
	}

	_, err = awxService.PostSurvey(d.Get("job_template_id").(int), survey, map[string]string{})
	if err != nil {
		log.Printf("Fail to add surveys %v", err)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to add surveys",
			Detail:   fmt.Sprintf("JobTemplate with id %d, failed to add survey: %s", d.Get("job_template_id").(int), err.Error()),
		})
		return diags
	}
	d.SetId(strconv.Itoa(d.Get("job_template_id").(int)))
	return resourceSurveyRead(ctx, d, m)
}

func getSurveySpecs(d *schema.ResourceData) ([]awx.SurveySpec, error) {
	surveySpecs := make([]awx.SurveySpec, 0)
	for _, input := range d.Get("spec").(*schema.Set).List() {
		var surveySpec awx.SurveySpec
		inputMap := input.(map[string]interface{})
		err := mapstructure.Decode(inputMap, &surveySpec)
		if err != nil {
			return nil, err
		}
		surveySpecs = append(surveySpecs, surveySpec)
	}
	return surveySpecs, nil
}

func resourceSurveyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return resourceSurveyCreate(ctx, d, m)
}

func resourceSurveyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*awx.AWX)
	awxService := client.JobTemplateService
	id, diags := convertStateIDToNummeric("Survey Read", d)
	if diags.HasError() {
		return diags
	}

	res, err := awxService.GetSurveyByJobTemplateId(id, make(map[string]string))
	if err != nil {
		return buildDiagNotFoundFail("job template", id, err)

	}

	d.Set("job_template_id", id)
	d, err = setSurveyResourceData(d, res)
	if err != nil {
		return buildDiagNotFoundFail("job template", id, err)

	}
	return nil
}

func setSurveyResourceData(d *schema.ResourceData, r *awx.Survey) (*schema.ResourceData, error) {
	d.Set("name", r.Name)
	d.Set("description", r.Description)
	surveySpecs := make([]map[string]interface{}, 0)
        for _, spec := range r.Spec {
		surveySpec := make(map[string]interface{}, 0)
		err := mapstructure.Decode(spec, &surveySpec)
		if err != nil {
			return nil, err
		}
		surveySpecs= append(surveySpecs, surveySpec)
	}
	err := d.Set("spec", surveySpecs)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func resourceSurveyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*awx.AWX)
	awxService := client.JobTemplateService
	id := d.Get("job_template_id").(int)

	_, err := awxService.DeleteSurvey(id)
	if err != nil {
		return buildDiagNotFoundFail("job template", id, err)

	}
	
	d.Set("name", nil)
	d.Set("description", nil)
	d.Set("spec", nil)
	d.SetId("")

	return nil
}
