package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Deleting a repository or a group only schedules it to be deleted for good seven days later: until
// then an Owner can restore it, and it keeps its path, so nothing new can take it. purge_on_destroy
// deletes it for good on destroy, as the API's permanently=true does, for a configuration that
// destroys and creates the same names again.

// repositoryOverride adds archived, set with the API's archive and unarchive, and purge_on_destroy.
// An archived repository takes no change to its settings, so unarchiving goes before an update,
// and archiving after one.
var repositoryOverride = resourceOverride{
	Schema: func(s *schema.Schema) {
		s.Attributes["archived"] = schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Description:         "Whether the repository is archived: read-only. It can still be read, cloned, starred and forked, and its access, visibility and webhooks still change, but nothing changes what it holds or its settings, and its running pipelines stop. Only Owners archive and unarchive.",
			MarkdownDescription: "Whether the repository is archived: read-only. It can still be read, cloned, starred and forked, and its access, visibility and webhooks still change, but nothing changes what it holds or its settings, and its running pipelines stop. Only Owners archive and unarchive.",
		}
		s.Attributes["purge_on_destroy"] = purgeOnDestroy("repository")
	},
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		object["archived"] = object["archived_at"] != nil
		return nil
	},
	Send: func(ctx context.Context, call *apiCall, send func() (any, error)) (any, error) {
		switch call.Operation {
		case "create":
			out, err := send()
			// What only an update takes is set by an update right after this, which archives after.
			if err != nil || !plannedTrue(call, "archived") || plansAny(call, resourceAPIs["repository"].UpdateOnly) {
				return out, err
			}
			created, _ := out.(map[string]any)
			id, _ := created["repository_id"].(string)
			if id == "" {
				return out, errors.New("the API didn't say which repository it created, so it couldn't be archived")
			}
			return setArchived(ctx, call.Client, "/repositories/"+url.PathEscape(id), true)
		case "update":
			current, err := call.Client.Get(ctx, call.Path, nil, nil)
			if err != nil {
				return nil, err
			}
			object, _ := current.(map[string]any)
			archived := object["archived_at"] != nil
			want, _ := call.Planned("archived")
			switch {
			case archived && want == false:
				if _, err := setArchived(ctx, call.Client, call.Path, false); err != nil {
					return nil, err
				}
				return send()
			case !archived && want == true:
				if _, err := send(); err != nil {
					return nil, err
				}
				return setArchived(ctx, call.Client, call.Path, true)
			case archived && unchanged(call):
				// Nothing the update takes changes, like when only purge_on_destroy does: an
				// archived repository would refuse it all the same.
				return nil, nil
			}
			return send()
		case "delete":
			if _, err := send(); err != nil {
				return nil, err
			}
			return nil, purge(ctx, call)
		}
		return send()
	},
}

// groupOverride adds purge_on_destroy.
var groupOverride = resourceOverride{
	Schema: func(s *schema.Schema) {
		s.Attributes["purge_on_destroy"] = purgeOnDestroy("group")
	},
	Send: func(ctx context.Context, call *apiCall, send func() (any, error)) (any, error) {
		out, err := send()
		if err != nil || call.Operation != "delete" {
			return out, err
		}
		return nil, purge(ctx, call)
	},
}

func purgeOnDestroy(kind string) schema.BoolAttribute {
	description := "Delete the " + kind + " for good on destroy. Otherwise destroying it schedules its deletion: it's gone for everyone at once, but an Owner can restore it for seven days, and until then it keeps its path, so no new " + kind + " can take it."
	if kind == "group" {
		description += " A group is deleted for good only once what it held is: set it on the repositories and subgroups in it too."
	}
	return schema.BoolAttribute{Optional: true, Description: description, MarkdownDescription: description}
}

// purge deletes for good what the delete call just scheduled for deletion, when the state's
// purge_on_destroy asks for it. If that fails, the delete still happened: it's a warning.
func purge(ctx context.Context, call *apiCall) error {
	if want, _ := call.Prior("purge_on_destroy"); want != true {
		return nil
	}
	err := call.Client.Do(ctx, http.MethodDelete, call.Path, url.Values{"permanently": {"true"}}, nil, nil)
	switch {
	case err == nil:
		return nil
	case HasCode(err, "GROUP_NOT_EMPTY"):
		return &warning{summary: "not for good: it still holds deleted repositories or subgroups. Set purge_on_destroy on them too, or it's deleted for good with them in seven days", err: err}
	default:
		return &warning{summary: "not for good: an Owner can restore it for seven days, and it keeps its path until then", err: err}
	}
}

// setArchived archives or unarchives the repository at path, and returns it.
func setArchived(ctx context.Context, client *Client, path string, archived bool) (any, error) {
	action := "/unarchive"
	if archived {
		action = "/archive"
	}
	var out any
	err := client.Do(ctx, http.MethodPost, path+action, nil, nil, &out)
	return out, err
}

func plannedTrue(call *apiCall, attribute string) bool {
	value, _ := call.Planned(attribute)
	return value == true
}

// plansAny reports whether the plan sets any of the attributes.
func plansAny(call *apiCall, attributes []string) bool {
	for _, attribute := range attributes {
		if _, ok := call.Planned(attribute); ok {
			return true
		}
	}
	return false
}

// unchanged reports whether an update's body holds only what the object has already: its fields
// are its attributes' JSON.
func unchanged(call *apiCall) bool {
	for field, value := range call.Body {
		prior, ok := call.Prior(field)
		if !ok {
			return false
		}
		sent, errSent := json.Marshal(value)
		held, errHeld := json.Marshal(prior)
		if errSent != nil || errHeld != nil || string(sent) != string(held) {
			return false
		}
	}
	return true
}
