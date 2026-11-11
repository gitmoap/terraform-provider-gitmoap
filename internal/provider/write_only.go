package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Write-only twins. Each sensitive argument, <name>, at the root or in an object, gets two more:
// <name>_wo, write-only, which Terraform 1.11 and later send without keeping in the plan or the
// state, and <name>_wo_version, a number to change to send <name>_wo again, since Terraform can't
// tell when a value it doesn't keep changes. <name> still works, for older versions of Terraform.

// addWriteOnly adds the twins of the attributes at paths to a schema.
func addWriteOnly(s *schema.Schema, paths []string) error {
	for _, dotted := range paths {
		parent, name, nested := strings.Cut(dotted, ".")
		if !nested {
			if err := addTwins(s.Attributes, parent); err != nil {
				return err
			}
			continue
		}
		object, ok := s.Attributes[parent].(schema.SingleNestedAttribute)
		if !ok {
			return fmt.Errorf("%s isn't an object, so %s can't get a write-only twin", parent, dotted)
		}
		// The type generated for the object doesn't know the twins, and nothing here uses it. And an
		// object holding a write-only attribute can't be computed: it's set from the configuration.
		object.CustomType = nil
		if object.Computed {
			object.Computed, object.Optional = false, !object.Required
		}
		if err := addTwins(object.Attributes, name); err != nil {
			return err
		}
		s.Attributes[parent] = object
	}
	return nil
}

func addTwins(attributes map[string]schema.Attribute, name string) error {
	original, ok := attributes[name].(schema.StringAttribute)
	if !ok {
		return fmt.Errorf("%s isn't a string, so it can't get a write-only twin", name)
	}
	twin, version := name+"_wo", name+"_wo_version"
	attributes[twin] = schema.StringAttribute{
		Optional:            true,
		WriteOnly:           true,
		Sensitive:           true,
		Description:         fmt.Sprintf("Sets %s without keeping it in Terraform's plan or state: write-only, with Terraform 1.11 and later. Change %s to send it again.", name, version),
		MarkdownDescription: fmt.Sprintf("Sets `%s` without keeping it in Terraform's plan or state: write-only, with Terraform 1.11 and later. Change `%s` to send it again.", name, version),
		Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName(name))},
	}
	attributes[version] = schema.Int64Attribute{
		Optional:            true,
		Description:         fmt.Sprintf("A number to change when %s changes, to send it again: Terraform can't tell, since it doesn't keep it.", twin),
		MarkdownDescription: fmt.Sprintf("A number to change when `%s` changes, to send it again: Terraform can't tell, since it doesn't keep it.", twin),
		Validators:          []validator.Int64{int64validator.AlsoRequires(path.MatchRelative().AtParent().AtName(twin))},
	}
	if original.Required {
		original.Required, original.Optional = false, true
		original.Validators = append(original.Validators, stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName(twin)))
	}
	original.Description = strings.TrimSpace(original.Description + " Or set " + twin + ", to keep it out of the state.")
	original.MarkdownDescription = strings.TrimSpace(original.MarkdownDescription + " Or set `" + twin + "`, to keep it out of the state.")
	attributes[name] = original
	return nil
}

// withWriteOnly returns values for a request body: each twin's value, from the configuration, in
// the attribute it stands for, and the twins themselves left out.
func withWriteOnly(values, config tftypes.Value, paths []string) tftypes.Value {
	for _, dotted := range paths {
		if secret, ok := attributeAt(config, dotted+"_wo"); ok && secret.IsKnown() && !secret.IsNull() {
			values = withAttributeAt(values, dotted, secret)
		}
		for _, twin := range []string{dotted + "_wo", dotted + "_wo_version"} {
			if current, ok := attributeAt(values, twin); ok {
				values = withAttributeAt(values, twin, tftypes.NewValue(current.Type(), nil))
			}
		}
	}
	return values
}
