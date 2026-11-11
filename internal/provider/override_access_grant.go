package provider

import (
	"context"
	"errors"
)

// accessGrantOverride serves group and repository access grants. The API gives a role to a list of
// subjects and answers with the grants it made, changes a grant and answers with it, and reads
// grants back from who has access here, where a grant's subject and role are summaries. A grant
// resource is one subject's grant: in the spec the generator reads, creating one takes a subject,
// so the subject goes back into a list here; and the summaries become the references the rest of
// the API takes, {"principal": "<id>"} and {"builtin": "developer"}.
var accessGrantOverride = resourceOverride{
	Request: func(ctx context.Context, call *apiCall) error {
		if call.Operation == "create" {
			call.Body["subjects"] = []any{call.Body["subject"]}
			delete(call.Body, "subject")
		}
		return nil
	},
	Response: func(ctx context.Context, operation string, object map[string]any) error {
		switch operation {
		case "create":
			grants, _ := object["grants"].([]any)
			grant, _ := firstOf(grants).(map[string]any)
			if grant == nil {
				return errors.New("the API made no grant: the subject may hold this role here already, in a grant to import instead")
			}
			replaceWith(object, grant)
		case "update":
			if grant, ok := object["grant"].(map[string]any); ok {
				replaceWith(object, grant)
			}
		}
		if subject, ok := object["subject"].(map[string]any); ok {
			object["subject"] = subjectReference(subject)
		}
		if role, ok := object["role"].(map[string]any); ok {
			object["role"] = map[string]any{"builtin": role["builtin"], "custom": role["custom"]}
		}
		return nil
	},
}

// subjectReference turns {"principal": {"id": …, "username": …}} into {"principal": "<id>"}, and the
// same for a team.
func subjectReference(subject map[string]any) map[string]any {
	reference := map[string]any{}
	for _, kind := range []string{"principal", "team"} {
		switch value := subject[kind].(type) {
		case map[string]any:
			reference[kind] = value["id"]
		case string:
			reference[kind] = value
		}
	}
	return reference
}

func firstOf(items []any) any {
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func replaceWith(object, replacement map[string]any) {
	clear(object)
	for key, value := range replacement {
		object[key] = value
	}
}
