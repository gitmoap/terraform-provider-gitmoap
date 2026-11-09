package provider

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// wikiLocks holds a lock for each wiki, by its path.
var wikiLocks sync.Map

// wikiPageOverride serves group and repository wiki pages. A wiki is a git repository: updating or
// deleting a page takes the head commit the wiki was read at, as expected_head_sha, and the API
// refuses it if the wiki has moved since. Every write moves it, so the writes to one wiki take turns,
// and each reads the head right before it goes. The page's head_sha is the whole wiki's, so it's
// left out of the state: a change to any other page would make it stale.
var wikiPageOverride = resourceOverride{
	Schema: func(s *schema.Schema) {
		delete(s.Attributes, "head_sha")
	},
	Send: func(ctx context.Context, call *apiCall, send func() (any, error)) (any, error) {
		if call.Operation == "read" {
			return send()
		}
		wiki, _, _ := strings.Cut(call.Path, "/wiki/")
		lock, _ := wikiLocks.LoadOrStore(wiki, &sync.Mutex{})
		lock.(*sync.Mutex).Lock()
		defer lock.(*sync.Mutex).Unlock()

		if call.Operation == "update" || call.Operation == "delete" {
			page, err := call.Client.Get(ctx, call.Path, nil, nil)
			if err != nil {
				return nil, err
			}
			object, _ := page.(map[string]any)
			if call.Body == nil {
				call.Body = map[string]any{}
			}
			call.Body["expected_head_sha"] = object["head_sha"]
		}
		return send()
	},
}
