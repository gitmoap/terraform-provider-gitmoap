package provider

// The operations behind each resource and data source. scripts/generate_provider.py writes them to
// api_gen.go from provider_api.json. Attribute paths are attribute names joined with dots.

type operation struct {
	Method string
	Path   string
	// Body lists what the request body takes, as attribute -> JSON field.
	Body map[string]string
	// Query lists the query parameters a data source sends, as parameter -> attribute.
	Query map[string]string
	// Items is the field holding the items of a list read's response, when the response isn't
	// the list itself.
	Items string
	// Pages is how to read every page of a list, if it has pages.
	Pages *pages
}

type pages struct {
	Param string // the query parameter that takes the previous page's next_token
	Items string // the array each page holds
	Limit int64  // the largest page the API serves, if it says
}

type resourceAPI struct {
	Create operation
	Read   operation
	Update *operation
	Delete *operation
	// Params lists the attribute behind each path parameter.
	Params map[string]string
	// Identity lists the attributes that identify the object, in path order: what import takes.
	Identity []string
	// Match is set when Read returns a list: an item is the object when the value at each JSON
	// path in Match equals its attribute.
	Match map[string]string
	// Replace lists the attributes only create sets, so changing them forces replacement.
	Replace []string
	// Renames lists the attributes named differently from their JSON field.
	Renames map[string]string
	// Transforms lists the attributes whose values the provider translates: see values.go.
	Transforms map[string]string
	// UpdateOnly lists the attributes only the update operation takes, set by an update after
	// creating.
	UpdateOnly []string
	// WriteOnly lists the sensitive attributes that get a write-only twin, <name>_wo, sent without
	// being kept in the state, and <name>_wo_version, which sends it again when it changes.
	WriteOnly []string
}

type dataSourceAPI struct {
	Read   operation
	Params map[string]string
	// Root is the attribute that holds the response when it's an array.
	Root       string
	Renames    map[string]string
	Transforms map[string]string
}
