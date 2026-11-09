#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = ["pyyaml"]
# ///
"""Generate generator_config.yml and provider_code_spec.json from the gitmoap OpenAPI spec.

    python3 scripts/generate_generator_config.py      # writes both files
    python3 scripts/generate_generator_config.py -v   # ...and lists every skip and warning

Run this script rather than tfplugingen-openapi itself: it runs the generator on a corrected copy of
the spec, then corrects what the generator writes. In order:

  1. Merges the allOf schemas listed in FLATTEN_ALLOF, in a copy of the spec. The generator can't map
     an allOf with several parts, and drops the whole body that uses one.
  2. Works out the resources and data sources, and writes generator_config.yml.
  3. Runs tfplugingen-openapi with that config on the corrected spec.
  4. Makes the parameters listed in CREATE_PATH_PARAMS required attributes that force replacement.
     The generator only maps the read operation's parameters, and lets a response field of the same
     name win, so a parent ID like group_id comes out optional, read-only, or missing.
  5. Makes the required attributes inside an optional, computed nested attribute optional and
     computed, so Terraform accepts the values the API fills in. See fix_api_filled_nested. And
     marks the attributes named in SECRET_NAMES sensitive.
  6. Renames the nested attributes listed in RENAME_NESTED, and writes provider_code_spec.json.
     tfplugingen-framework names a nested attribute's Go types after the attribute alone, so two
     nested attributes with the same name in one resource or data source (updated_by and
     history.updated_by) both declare UpdatedByType, and the package doesn't compile.
  7. Writes provider_api.json, which scripts/generate_provider.py turns into Go: for each resource
     and data source, its operations, the request body fields each takes, the attribute behind
     each path parameter, how a list read finds its object, and which attributes only create can
     set. A parameter whose attribute can't be found by name goes in PATH_PARAM_ATTRIBUTES.

The fixes apply only to the cases listed in those constants. A new endpoint with the same
pattern is reported as a problem along with the entry to add, and so is a listed entry that no
longer matches anything. The script exits with 1 while there are problems, but writes both files.

Resources are detected from the shape of the paths:
    - POST /things            + GET /things/{id}            collection and item
    - POST /owners/{o}/things + GET /things/{id}            item lives outside its owner
    - POST /things            + GET /things (no item GET)   read has to filter the list
    - PUT  /things/{key}      + GET /things/{key}           upsert by key
    - PUT  /things/{key}      + GET /things                 upsert by key, read from the list
    - PUT|PATCH /settings     + GET /settings               singleton
A resource that can be created but not deleted is skipped: it records an action, like publishing.

Data sources come from every remaining GET that returns JSON: item, list and singleton.

schema.ignores lists whatever tfplugingen-openapi still can't map. One unsupported property makes
the generator drop the *whole* body it sits in, logging only a warning. `Spec.map_body` mirrors the
generator's rules (internal/mapper/oas in v0.3.0); re-check it against the source when upgrading.
The script fails if the generator drops anything, which catches drift.

schema.attributes.aliases only merges a resource's own ID parameter into its body field
(repo_id -> repository_id). Parent IDs are handled by CREATE_PATH_PARAMS instead.

Hand-written changes to the config go in the overrides file. It has the same layout as the config
and is merged over it last. Lists are unioned and null removes a key:

    resources:
      repository_tag: null                # drop a generated resource
      repository:
        schema:
          attributes:
            overrides:
              visibility: {description: Who can see the repository.}
"""

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
import tempfile
from collections import defaultdict
from collections.abc import Iterator
from dataclasses import dataclass, field
from pathlib import Path

import yaml

# --------------------------------------------------------------------------------------------------
# Policy: the choices a person would want to revisit. Everything below this section is mechanics.
# --------------------------------------------------------------------------------------------------

PROVIDER_NAME = "gitmoap"
REPO_ROOT = Path(__file__).resolve().parent.parent

# Paths that don't describe infrastructure someone would declare in Terraform. Matched with
# re.search against the path, and applied to both resources and data sources. New API areas are
# included by default, so a new endpoint shows up unless it matches something here.
EXCLUDED_PATHS = {
    r"^/(\.well-known|auth|oauth|scim|packages)(/|$)": "protocol endpoint",
    r"^/admin(/|$)": "platform administration",
    r"^/(user|notifications)(/|$)": "scoped to the calling user",
    r"^/(legal|contact|feedback|search|resolve|scopes|capabilities|marketplace|billing)(/|$)": "platform utility",
    r"/(issues|merge-requests|pipeline-runs|ai-sessions|ritual-runs|messages|reactions|anchors|code-todos)(/|$)": "day-to-day workflow object",
    r"/(commits|tree|tree-paths|blob|compare|diff|diff-file|diff-files|file-lines|grep|code-search|search|contributors|commit-authors|languages|codeowners|pipeline-diff)(/|$)": "git content browsing",
    r"/specs(/|$)": "git content browsing",
    r"/(activity|audit-log|insights|usage|egress-usage|people|deliveries|trigger-evaluations|fork-inventory|joinable-iterations)(/|$)": "reporting or activity feed",
    r"/(contributions|burndown|velocity|digests|review-queue|ritual-calendar|findings|backlinks|revisions|graph|overview|upgrade-impact|usages|drift|explain|review|archive|watch|export)(/|$)": "reporting or activity feed",
    r"/(attachments|avatar|star|transfer|[a-z-]+-preview|pipelines-preview)(/|$)": "action or binary upload",
    r"/(billing|limits|limit-increase|limit-increase-requests)(/|$)": "billing and quotas",
}

# Abbreviations in path parameters, so repo_id can be matched to a repository_id field.
ABBREVIATIONS = {"repo": "repository", "mr": "merge_request", "sa": "service_account", "org": "organization"}

# Handled by the provider rather than exposed as attributes of a list data source.
PAGINATION_FIELDS = {"limit", "next_token", "cursor", "next_cursor", "page", "per_page", "page_size", "offset"}

IRREGULAR_SINGULARS = {"statuses": "status", "aliases": "alias", "criteria": "criterion"}

# ReservedResourceAttributeNames in terraform-plugin-framework: a provider using one of these as a
# root attribute fails validation at startup.
RESERVED_ROOT_ATTRIBUTES = {"connection", "count", "depends_on", "for_each", "lifecycle", "provider", "provisioner"}

# Known cases of the three fixes. Only these get fixed: the script reports new cases, with the entry
# to add here, and entries that stopped matching.

# Component schemas built with allOf from several parts, merged into one object in the spec the
# generator reads.
FLATTEN_ALLOF = {
    "ApprovalPolicy",
    "BlueprintView",
    "ComponentListItem",
    "ComponentVersionResponse",
    "MilestoneView",
    "PackageDetail",
    "RepositoryGroupView",
    "RepositoryView",
    "ResolvedConnection",
    "ScimTokenCreated",
    "WikiPageDetail",
}

# Create path parameters, by resource, made required attributes that force replacement.
CREATE_PATH_PARAMS = {
    "ai_agent_memory": ["agent_id"],
    "catalog_scorecard": ["group_id"],
    "catalog_environment_variable": ["environment_id", "name"],
    "catalog_service": ["group_id"],
    "group_ai_org_conventions": ["group_id"],
    "group_ai_org_squad_memory": ["group_id", "squad_key"],
    "group_approval_policy": ["group_id"],
    "group_branch_rule": ["group_id"],
    "group_catalog_environment": ["group_id"],
    "group_fork_settings": ["group_id"],
    "group_issue_blueprint": ["group_id"],
    "group_issue_settings": ["group_id"],
    "group_iteration": ["group_id"],
    "group_iteration_cadence": ["group_id"],
    "group_label": ["group_id"],
    "group_member": ["group_id"],
    "group_milestone": ["group_id"],
    "group_mr_settings": ["group_id"],
    "group_pipeline_variable": ["group_id", "name"],
    "group_provider_connection": ["group_id"],
    "group_provider_policy": ["group_id"],
    "group_push_rules": ["group_id"],
    "group_scim_token": ["group_id"],
    "group_service_account": ["group_id"],
    "group_sso_connection": ["group_id"],
    "group_sso_domain": ["group_id"],
    "group_sso_invite": ["group_id"],
    "group_tag_rule": ["group_id"],
    "group_webhook": ["group_id"],
    "group_wiki": ["group_id"],
    "group_wiki_page": ["group_id"],
    "provider_listing": ["provider_id"],
    "repository_approval_policy": ["repo_id"],
    "repository_branch": ["repo_id"],
    "repository_branch_rule": ["repo_id"],
    "repository_catalog_environment": ["repo_id"],
    "repository_fork": ["repo_id"],
    "repository_fork_settings": ["repo_id"],
    "repository_issue_blueprint": ["repo_id"],
    "repository_issue_settings": ["repo_id"],
    "repository_label": ["repo_id"],
    "repository_member": ["repo_id"],
    "repository_milestone": ["repo_id"],
    "repository_mr_settings": ["repo_id"],
    "repository_pipeline_variable": ["repo_id", "name"],
    "repository_provider_connection": ["repo_id"],
    "repository_push_rules": ["repo_id"],
    "repository_release": ["repo_id"],
    "repository_release_asset": ["repo_id", "tag_name"],
    "repository_service_account": ["repo_id"],
    "repository_tag": ["repo_id"],
    "repository_tag_rule": ["repo_id"],
    "repository_webhook": ["repo_id"],
    "repository_wiki": ["repo_id"],
    "repository_wiki_page": ["repo_id"],
    "role": ["group_id"],
    "service_account_token": ["sa_id"],
    "team": ["group_id"],
    "team_member": ["team_id", "user_id"],
}

# Nested attributes renamed so their generated Go types don't clash, by resource or data source:
# attribute path -> new name. Paths use the generated names joined with dots, and list items add no
# step, as in schema.ignores. The shallowest attribute of a name keeps it. The provider code maps
# the renamed attribute to its API field by hand.
RENAME_NESTED: dict[str, dict[str, str]] = {
    "data source ai_agent_memory": {"history.updated_by": "history_updated_by"},
    "data source group_access": {"entries.source.at": "source_at"},
    "data source group_ai_org_conventions": {"history.updated_by": "history_updated_by"},
    "data source group_ai_org_squad_memory": {"history.updated_by": "history_updated_by"},
    "data source group_approval_policy": {"rules.defined_at": "rules_defined_at", "rules_here.defined_at": "rules_here_defined_at"},
    "data source provider_submission": {"manifest.capabilities": "manifest_capabilities", "mirror_result.capabilities": "mirror_result_capabilities"},
    "data source repository_access": {"entries.source.at": "source_at"},
    "data source repository_approval_policy": {"rules.defined_at": "rules_defined_at", "rules_here.defined_at": "rules_here_defined_at"},
    "data source repository_pipeline": {"diagnostics.warnings.target": "warnings_target"},
    "data source repository_pipeline_component_version": {"body.outputs.source": "outputs_source"},
    "data source repository_pipeline_file": {
        "diagnostics.fix.location": "fix_location",
        "diagnostics.fix.location.end": "location_end",
        "diagnostics.fix.location.start": "location_start",
        "diagnostics.related.end": "related_end",
        "diagnostics.related.start": "related_start",
    },
    "data source repository_pipelines": {"items.diagnostics.warnings.target": "warnings_target"},
    "resource ai_agent_memory": {"history.updated_by": "history_updated_by"},
    "resource group_ai_org_conventions": {"history.updated_by": "history_updated_by"},
    "resource group_ai_org_squad_memory": {"history.updated_by": "history_updated_by"},
    "resource group_approval_policy": {"rules.defined_at": "rules_defined_at", "rules_here.defined_at": "rules_here_defined_at"},
    "resource repository_approval_policy": {"rules.defined_at": "rules_defined_at", "rules_here.defined_at": "rules_here_defined_at"},
}

# Path parameters whose attribute the script can't find by name, by resource or data source:
# parameter -> attribute path (dots for nested attributes). The rest match an attribute of the same
# name, their alias, the name with its abbreviation expanded (repo_id -> repository_id), or the
# resource's own field (label_key -> key on a label).
PATH_PARAM_ATTRIBUTES: dict[str, dict[str, str]] = {
    "resource service_account_token": {"token_id": "access_token.access_token_id"},
}

# --------------------------------------------------------------------------------------------------
# Reading the spec, mirroring how tfplugingen-openapi interprets it
# --------------------------------------------------------------------------------------------------

SCALARS = {"string", "integer", "number", "boolean"}
STRINGABLE = {"integer", "number", "boolean"}
COMPONENT_PREFIX = "#/components/schemas/"


class UnsupportedSchema(Exception):
    """The generator returns an error for this schema."""

    def __init__(self, message: str, allof: str | None = None):
        super().__init__(message)
        # The component schema whose allOf is the cause, when there is one with a name.
        self.allof = allof


def oas_type(schema: dict) -> str:
    """Mirrors retrieveType in build.go."""
    types = schema.get("type")
    types = [] if types is None else [types] if isinstance(types, str) else types
    if not types:
        if schema.get("properties"):
            return "object"
        raise UnsupportedSchema("no type")
    if len(types) == 1:
        return types[0]
    if len(types) == 2:
        first, second = types
        if first == "null":
            return second
        if second == "null":
            return first
        if first == "string" and second in STRINGABLE:
            return first
        if second == "string" and first in STRINGABLE:
            return second
    raise UnsupportedSchema(f"multi-type {types}")


def tf_identifier(name: str) -> str:
    """Mirrors TerraformIdentifier in framework_identifier.go."""
    name = re.sub(r"[^a-zA-Z0-9_]+", "", name)
    name = re.sub(r"^\d+", "", name)
    name = re.sub(r"([a-z])([A-Z])", r"\1_\2", name)
    return name.lower()


@dataclass
class BodyMapping:
    """How the generator will map a request or response body."""

    root: str | None  # why none of it maps, if it doesn't
    ignores: dict[str, str]  # what to ignore so the rest maps, and why
    allof: set[str]  # unmerged allOf schemas behind any of the above


class Spec:
    def __init__(self, doc: dict):
        self.doc = doc
        self.paths: dict = doc["paths"]
        self.schemas: dict = doc.get("components", {}).get("schemas", {})
        # The original allOf parts of each schema merge_allof replaced, by id of the merged schema.
        self.merged_from: dict[int, list] = {}

    def resolve(self, node) -> tuple[dict, str | None]:
        """Follow $refs. Returns the schema, and the name of the component schema it is, if any."""
        name, seen = None, set()
        while isinstance(node, dict) and "$ref" in node:
            ref = node["$ref"]
            if ref in seen:
                raise UnsupportedSchema(f"circular reference {ref}")
            seen.add(ref)
            name = ref.removeprefix(COMPONENT_PREFIX) if ref.startswith(COMPONENT_PREFIX) else None
            node = self.doc
            for part in ref.removeprefix("#/").split("/"):
                node = node[part.replace("~1", "/").replace("~0", "~")]
        return node, name

    def deref(self, node):
        return self.resolve(node)[0]

    def effective(self, node, seen=()) -> dict:
        """The schema the generator ends up using for a node. Mirrors buildSchemaProxy in build.go."""
        schema, name = self.resolve(node)
        if id(schema) in seen:
            raise UnsupportedSchema("circular composition")
        seen = (*seen, id(schema))
        for keyword in ("anyOf", "oneOf"):
            if schema.get(keyword):
                if len(schema[keyword]) != 2:
                    raise UnsupportedSchema(f"{keyword} with {len(schema[keyword])} subschemas")
                return self._pick_from_pair(*schema[keyword], seen)
        if schema.get("allOf"):
            if len(schema["allOf"]) != 1:
                raise UnsupportedSchema(f"allOf with {len(schema['allOf'])} subschemas", allof=name)
            return self.effective(schema["allOf"][0], seen)
        return schema

    def _pick_from_pair(self, one, other, seen) -> dict:
        """Mirrors getMultiTypeSchema in build.go."""
        first, second = self.effective(one, seen), self.effective(other, seen)
        first_type, second_type = oas_type(first), oas_type(second)
        if first_type == "null":
            return second
        if second_type == "null":
            return first
        if first_type == "string" and second_type in STRINGABLE:
            return first
        if second_type == "string" and first_type in STRINGABLE:
            return second
        raise UnsupportedSchema(f"multi-type [{first_type} {second_type}]")

    def merge_allof(self, name: str) -> None:
        """Replace component schema `name`, built with allOf, by one object holding its parts' properties."""
        schema = self.schemas[name]
        properties, required = self.allof_members(schema)
        merged = {key: value for key, value in schema.items() if key != "allOf"}
        merged.update(type="object", properties=properties)
        if required:
            merged["required"] = required
        self.schemas[name] = merged
        self.merged_from[id(merged)] = schema["allOf"]

    def allof_members(self, schema: dict, seen=()) -> tuple[dict, list]:
        """Properties and required names of an allOf, if each part is an object with plain properties."""
        properties = dict(schema.get("properties") or {})
        required = list(schema.get("required") or [])
        for raw in schema["allOf"]:
            part, name = self.resolve(raw)
            label = name or "an inline part"
            if id(part) in seen:
                raise UnsupportedSchema(f"{label} includes itself")
            if part.get("allOf"):
                part_properties, part_required = self.allof_members(part, (*seen, id(part)))
            elif part.get("oneOf") or part.get("anyOf"):
                raise UnsupportedSchema(f"{label} uses {'oneOf' if part.get('oneOf') else 'anyOf'}")
            elif oas_type(part) != "object":
                raise UnsupportedSchema(f"{label} isn't an object")
            else:
                part_properties, part_required = part.get("properties") or {}, part.get("required") or []
            for prop, value in part_properties.items():
                if properties.get(prop, value) != value:
                    raise UnsupportedSchema(f"two parts define {prop} differently")
                properties[prop] = value
            required += [prop for prop in part_required if prop not in required]
        return properties, required

    def map_body(self, node) -> BodyMapping:
        """Find what must be ignored for this body schema to map.

        A root reason can't be fixed with an ignore. Array items and map values add no segment to a
        location, as in GetIgnoresForNested, so an unsupported item schema means ignoring the array.

        Free-form objects are the one deliberate difference from the generator, which accepts them.
        They would become a nested attribute with no attributes, unable to hold any value.
        """
        allof: set[str] = set()

        def visit(value, location: tuple, ancestors: frozenset) -> tuple[str | None, dict[str, str]]:
            try:
                schema = self.effective(value)
                kind = oas_type(schema)
            except UnsupportedSchema as error:
                if error.allof:
                    allof.add(error.allof)
                return str(error), {}
            if id(schema) in ancestors:
                return "circular reference", {}
            ancestors = ancestors | {id(schema)}
            if kind in SCALARS:
                return None, {}
            if kind == "array":
                items = schema.get("items")
                if not isinstance(items, dict):
                    return "array without an items schema", {}
                return visit(items, location, ancestors)
            if kind != "object":
                return f"type {kind}", {}
            if isinstance(schema.get("additionalProperties"), dict):
                return visit(schema["additionalProperties"], location, ancestors)
            properties = schema.get("properties") or {}
            if not properties:
                return "free-form object", {}
            problems, unusable = {}, 0
            for name, prop in properties.items():
                reason, nested = visit(prop, (*location, name), ancestors)
                if reason:
                    unusable += 1
                    problems[".".join((*location, name))] = reason
                problems.update(nested)
            if unusable == len(properties):
                return "no supported properties", {}
            return None, problems

        root, ignores = visit(node, (), frozenset())
        return BodyMapping(root, ignores, allof)

    def root_properties(self, node) -> dict:
        """Top-level properties of a body, or nothing if the generator can't read its root."""
        try:
            schema = self.effective(node)
        except UnsupportedSchema:
            return {}
        return schema.get("properties") or {}

    def composed_properties(self, node) -> set[str]:
        """Root property names of a schema, across all its allOf parts."""
        schema = self.deref(node) if node else {}
        names = set(schema.get("properties") or {})
        for part in schema.get("allOf") or []:
            names |= self.composed_properties(part)
        return names

    def locate(self, node, location: str) -> bool | None:
        """Whether a location is required all the way from the root of a body schema, or None if
        it isn't there."""
        required = True
        for part in location.split("."):
            try:
                schema = self.effective(node)
                while oas_type(schema) == "array" or isinstance(schema.get("additionalProperties"), dict):
                    schema = self.effective(schema.get("items") or schema["additionalProperties"])
            except (UnsupportedSchema, KeyError):
                return None
            properties = schema.get("properties") or {}
            if part not in properties:
                return None
            required = required and part in (schema.get("required") or [])
            node = properties[part]
        return required

    def _json_schema(self, container) -> dict | None:
        media = self.deref(container).get("content") or {}
        return (media.get("application/json") or {}).get("schema")

    def request_schema(self, op: dict) -> dict | None:
        return self._json_schema(op["requestBody"]) if op.get("requestBody") else None

    def response_schema(self, op: dict) -> dict | None:
        """Mirrors the status code priority of BuildSchemaFromResponse."""
        responses = op.get("responses") or {}
        for code in ("200", "201"):
            if code in responses:
                return self._json_schema(responses[code])
        for code in sorted(responses):
            if code.isdigit() and 200 <= int(code) <= 299:
                return self._json_schema(responses[code])
        return None

    def op(self, path: str, method: str) -> dict | None:
        return (self.paths.get(path) or {}).get(method)

    def parameters(self, path: str, method: str) -> dict[tuple[str, str], dict]:
        """Path-level and operation-level parameters, keyed by (location, name)."""
        merged = {}
        for raw in (self.paths[path].get("parameters") or []) + (self.op(path, method).get("parameters") or []):
            param = self.deref(raw)
            merged[(param["in"], param["name"])] = param
        return merged


# --------------------------------------------------------------------------------------------------
# Naming
# --------------------------------------------------------------------------------------------------


def segments(path: str) -> list[str]:
    return [segment for segment in path.split("/") if segment]


def is_param(segment: str) -> bool:
    return segment.startswith("{") and segment.endswith("}")


def is_item_path(path: str) -> bool:
    return is_param(segments(path)[-1])


def path_params(path: str) -> list[str]:
    return [segment[1:-1] for segment in segments(path) if is_param(segment)]


def snake(text: str) -> str:
    text = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", text)
    return re.sub(r"[^A-Za-z0-9]+", "_", text).strip("_").lower()


def singular(word: str) -> str:
    for plural, single in IRREGULAR_SINGULARS.items():
        if word.endswith(plural):
            return word[: -len(plural)] + single
    if word.endswith("ies"):
        return word[:-3] + "y"
    if word.endswith(("sses", "ches", "shes", "xes")):
        return word[:-2]
    if word.endswith("s") and not word.endswith("ss"):
        return word[:-1]
    return word


def name_from_path(path: str) -> str:
    """/repositories/{repo_id}/webhooks/{id} -> repository_webhook, and without the {id}, _webhooks."""
    parts = segments(path)
    words = []
    for index, part in enumerate(parts):
        if is_param(part):
            continue
        following = parts[index + 1] if index + 1 < len(parts) else ""
        is_singular = is_param(following) or following.startswith("by-")
        words.append(snake(singular(part) if is_singular else part))
    return "_".join(words)


# --------------------------------------------------------------------------------------------------
# Discovery
# --------------------------------------------------------------------------------------------------


@dataclass
class Report:
    skipped: list[tuple[str, str]] = field(default_factory=list)
    warnings: list[tuple[str, str]] = field(default_factory=list)
    problems: list[tuple[str, str]] = field(default_factory=list)
    # Unmerged allOf schemas that cost something, and what they cost it to.
    allof_users: dict[str, set[str]] = field(default_factory=lambda: defaultdict(set))

    def skip(self, what: str, reason: str) -> None:
        self.skipped.append((what, reason))

    def warn(self, what: str, message: str) -> None:
        self.warnings.append((what, message))

    def problem(self, what: str, message: str) -> None:
        self.problems.append((what, message))

    def allof_cost(self, what: str, mapping: BodyMapping) -> None:
        for name in mapping.allof:
            self.allof_users[name].add(what)


@dataclass
class ResourceShape:
    name: str
    create: tuple[str, str]
    read: tuple[str, str]
    update: tuple[str, str] | None
    delete: tuple[str, str] | None
    read_is_list: bool = False


def exclusion_reason(path: str) -> str | None:
    for pattern, reason in EXCLUDED_PATHS.items():
        if re.search(pattern, path):
            return reason
    return None


def has_json_body(spec: Spec, path: str, method: str) -> bool:
    op = spec.op(path, method)
    return bool(op and spec.request_schema(op))


def write_method(spec: Spec, path: str) -> str | None:
    """The method that updates a path, if it takes a JSON body."""
    for method in ("put", "patch"):
        if has_json_body(spec, path, method):
            return method
    return None


def schema_identity(spec: Spec, node) -> set[int]:
    """A schema and the parts it composes, so that a *View wrapping a schema matches that schema."""
    schema = spec.deref(node) if node else None
    if not schema:
        return set()
    identity = {id(schema)}
    for part in schema.get("allOf") or spec.merged_from.get(id(schema), []):
        identity |= schema_identity(spec, part)
    return identity


def find_creators(spec: Spec, item_path: str) -> list[str]:
    """The POST paths that create what item_path reads."""
    parent = item_path.rsplit("/", 1)[0]
    if has_json_body(spec, parent, "post"):
        return [parent]
    # Created under an owner: POST /groups/{id}/teams + GET /teams/{id}. A shared suffix alone could
    # be a coincidence, so the POST also has to return the same thing the item GET does.
    read_op = spec.op(item_path, "get")
    item_identity = schema_identity(spec, spec.response_schema(read_op)) if read_op else set()
    creators = []
    for path in sorted(spec.paths):
        owner = path.removesuffix(parent)
        if path == parent or not path.endswith(parent) or not owner or not is_item_path(owner):
            continue
        if not has_json_body(spec, path, "post"):
            continue
        created = schema_identity(spec, spec.response_schema(spec.op(path, "post")))
        if not read_op or created & item_identity:
            creators.append(path)
    return creators


def discover_resources(spec: Spec, report: Report) -> list[ResourceShape]:
    shapes = []
    for path in sorted(spec.paths):
        reason = exclusion_reason(path)
        if reason:
            if write_method(spec, path) or has_json_body(spec, path, "post"):
                report.skip(f"resource at {path}", reason)
            continue

        updater = write_method(spec, path)
        update = (path, updater) if updater else None
        delete = (path, "delete") if spec.op(path, "delete") else None
        read = (path, "get") if spec.op(path, "get") else None

        if not is_item_path(path):
            if read and updater:
                shapes.append(ResourceShape(name_from_path(path), update, read, update, delete))
            continue

        creators = [c for c in find_creators(spec, path) if not exclusion_reason(c)]
        if creators:
            parent = path.rsplit("/", 1)[0]
            read_is_list = False
            if not read:
                if not spec.op(parent, "get"):
                    report.skip(f"resource at {path}", "no GET on the item or its collection")
                    continue
                read, read_is_list = (parent, "get"), True
            if not delete:
                # Nothing to run on destroy: this records a one-way action (submit, publish,
                # register), it doesn't manage an object.
                report.skip(f"resource at {path}", "can be created but not deleted")
                continue
            last = path.rsplit("/", 1)[1]
            for creator in creators:
                name = name_from_path(f"{creator}/{last}" if len(creators) > 1 else path)
                shapes.append(ResourceShape(name, (creator, "post"), read, update, delete, read_is_list))
        elif read and has_json_body(spec, path, "put"):
            shapes.append(ResourceShape(name_from_path(path), (path, "put"), read, (path, "put"), delete))
        elif has_json_body(spec, path, "put") and delete and spec.op(path.rsplit("/", 1)[0], "get"):
            parent_read = (path.rsplit("/", 1)[0], "get")
            shapes.append(ResourceShape(name_from_path(path), (path, "put"), parent_read, (path, "put"), delete, read_is_list=True))
        elif updater:
            report.skip(f"resource at {path}", "can be updated but not created through the API")
    return shapes


def item_alias(spec: Spec, path: str, method: str, properties: dict) -> dict[str, str]:
    """Merge a resource's own ID parameter into the body field holding the same value."""
    if not is_item_path(path):
        return {}
    parts = segments(path)
    param_name = parts[-1][1:-1]
    if param_name in properties or len(parts) < 2:
        return {}
    params = spec.parameters(path, method)
    param = params.get(("path", param_name))
    if not param:
        return {}
    collection = snake(singular(parts[-2]))
    if "_" in param_name:
        stem, suffix = param_name.rsplit("_", 1)
        candidates = [f"{ABBREVIATIONS.get(stem, stem)}_{suffix}", f"{collection}_{suffix}"]
        if suffix == "id":
            candidates.append("id")
    else:
        candidates = [f"{collection}_{param_name}"]
    taken = {name for location, name in params if location == "path"}
    try:
        param_type = oas_type(spec.effective(param.get("schema") or {}))
    except UnsupportedSchema:
        return {}
    for candidate in candidates:
        if candidate in properties and candidate not in taken:
            try:
                if oas_type(spec.effective(properties[candidate])) == param_type:
                    return {param_name: candidate}
            except UnsupportedSchema:
                continue
    return {}


def schema_section(ignores: set[str], aliases: dict[str, str]) -> dict:
    section: dict = {}
    if aliases:
        section["attributes"] = {"aliases": dict(sorted(aliases.items()))}
    if ignores:
        section["ignores"] = sorted(ignores)
    return section


def location_entry(location: tuple[str, str] | None) -> dict | None:
    return {"path": location[0], "method": location[1].upper()} if location else None


def build_resource(spec: Spec, shape: ResourceShape, report: Report) -> dict | None:
    what = f"resource {shape.name}"
    create_op = spec.op(*shape.create)
    read_op = spec.op(*shape.read)

    request = spec.request_schema(create_op)
    create_response = spec.response_schema(create_op)
    read_response = None if shape.read_is_list else spec.response_schema(read_op)

    request_mapping = spec.map_body(request)
    report.allof_cost(what, request_mapping)
    if request_mapping.root:
        report.skip(what, f"create request body can't be mapped: {request_mapping.root}")
        return None

    bodies = [request]
    unmappable = []
    ignores = dict(request_mapping.ignores)
    for label, body in (("create response", create_response), ("read response", read_response)):
        if body is None:
            continue
        mapping = spec.map_body(body)
        report.allof_cost(what, mapping)
        if mapping.root:
            unmappable.append((label, body, mapping.root))
            continue
        bodies.append(body)
        ignores.update(mapping.ignores)

    properties = {}
    for body in bodies:
        properties.update(spec.root_properties(body))

    # The generator drops a response it can't map, and only logs it. Usually another body carries
    # the same fields (a *View that extends the create response), so name only what's really lost.
    for label, body, root in unmappable:
        lost = sorted(spec.composed_properties(body) - set(properties))
        if lost:
            report.warn(what, f"{label} can't be mapped ({root}), so these fields are missing: {', '.join(lost)}")

    params = spec.parameters(*shape.read)
    # The generator adds every read query parameter as an attribute. On a resource those are request
    # options rather than state, unless a body field of the same name makes them real attributes.
    for location, name in params:
        if location == "query" and name not in properties:
            ignores[name] = "read query parameter"

    if shape.read_is_list:
        for name in spec.root_properties(spec.response_schema(read_op)):
            if name not in properties:
                ignores[name] = "list response field"

    aliases = {} if shape.read_is_list else item_alias(spec, *shape.read, properties)

    for name in {*properties, *(n for loc, n in params if loc == "path")}:
        attribute = aliases.get(name, name)
        if tf_identifier(attribute) in RESERVED_ROOT_ATTRIBUTES:
            ignores[attribute] = "reserved Terraform attribute name"

    blocked = []
    for location, reason in sorted(ignores.items()):
        required = spec.locate(request, location)
        if required:
            blocked.append(f"{location} ({reason})")
        elif required is not None:
            report.warn(what, f"optional request field {location} can't be set: {reason}")
    if blocked:
        report.skip(what, f"required request fields can't be mapped, so it could never be created: {'; '.join(blocked)}")
        return None

    entry = {"create": location_entry(shape.create), "read": location_entry(shape.read)}
    if shape.update:
        entry["update"] = location_entry(shape.update)
    if shape.delete:
        entry["delete"] = location_entry(shape.delete)
    section = schema_section(set(ignores), aliases)
    if section:
        entry["schema"] = section
    return entry


def build_data_sources(spec: Spec, report: Report) -> dict[str, dict]:
    data_sources: dict[str, dict] = {}
    for path in sorted(spec.paths):
        op = spec.op(path, "get")
        if not op:
            continue
        reason = exclusion_reason(path)
        if reason:
            report.skip(f"data source at {path}", reason)
            continue
        response = spec.response_schema(op)
        if not response:
            report.skip(f"data source at {path}", "no JSON response")
            continue
        name = name_from_path(path)
        mapping = spec.map_body(response)
        report.allof_cost(f"data source {name}", mapping)
        if mapping.root:
            report.skip(f"data source at {path}", f"response root can't be mapped: {mapping.root}")
            continue

        if name in data_sources:
            sys.exit(f"error: two data sources would be named {name}: {path} and {data_sources[name]['read']['path']}")

        properties = spec.root_properties(response)
        params = spec.parameters(path, "get")
        ignores = set(mapping.ignores)
        ignores |= {n for loc, n in params if loc == "query" and n in PAGINATION_FIELDS}
        ignores |= PAGINATION_FIELDS & set(properties)

        aliases = item_alias(spec, path, "get", properties)
        for param_name in {*properties, *(n for loc, n in params if loc in ("path", "query"))}:
            attribute = aliases.get(param_name, param_name)
            if tf_identifier(attribute) in RESERVED_ROOT_ATTRIBUTES:
                ignores.add(attribute)

        entry = {"read": location_entry((path, "get"))}
        section = schema_section(ignores, aliases)
        if section:
            entry["schema"] = section
        data_sources[name] = entry
    return data_sources


# --------------------------------------------------------------------------------------------------
# The fixes
# --------------------------------------------------------------------------------------------------


def merge_known_allof(spec: Spec, report: Report) -> int:
    merged = 0
    for name in sorted(FLATTEN_ALLOF):
        schema = spec.schemas.get(name)
        if schema is None or not schema.get("allOf"):
            state = "no longer exists" if schema is None else "no longer uses allOf"
            report.problem(f"schema {name}", f"is in FLATTEN_ALLOF but {state}. Fix: remove it from FLATTEN_ALLOF")
            continue
        try:
            spec.merge_allof(name)
            merged += 1
        except UnsupportedSchema as error:
            report.problem(f"schema {name}", f"is in FLATTEN_ALLOF but can't be merged any more: {error}")
    return merged


def report_unmerged_allof(spec: Spec, report: Report) -> None:
    for name, users in sorted(report.allof_users.items()):
        if name in FLATTEN_ALLOF:
            continue  # already reported by merge_known_allof
        cost = ", ".join(sorted(users))
        try:
            spec.allof_members(spec.schemas[name])
        except UnsupportedSchema as error:
            report.warn(f"schema {name}", f"uses allOf, which the generator can't map, and can't be merged ({error}). Costs: {cost}")
            continue
        report.problem(f"schema {name}", f"uses allOf, which the generator can't map. Costs: {cost}. Fix: add {json.dumps(name)} to FLATTEN_ALLOF")


ATTRIBUTE_TYPES = {"string": "string", "integer": "int64", "number": "number", "boolean": "bool"}
PLAN_MODIFIER_TYPES = {"string", "int64", "float64", "number", "bool"}


def attribute_type(attribute: dict) -> str:
    return next(key for key in attribute if key != "name")


def attribute_settability(attribute: dict) -> str | None:
    return attribute[attribute_type(attribute)].get("computed_optional_required")


def describe(attribute: dict | None) -> str:
    if attribute is None:
        return "missing"
    return {
        "computed": "read-only",
        "computed_optional": "optional",
        "optional": "optional",
        "required": "required but changing it doesn't force replacement",
    }.get(attribute_settability(attribute), "unexpected")


def requires_replace(type_key: str) -> dict:
    package = f"{type_key}planmodifier"
    return {
        "custom": {
            "imports": [{"path": f"github.com/hashicorp/terraform-plugin-framework/resource/schema/{package}"}],
            "schema_definition": f"{package}.RequiresReplace()",
        }
    }


def fix_create_path_params(spec: Spec, config: dict, code_spec: dict, report: Report) -> int:
    generated = {resource["name"]: resource for resource in code_spec.get("resources", [])}
    for name in sorted(set(CREATE_PATH_PARAMS) - set(config["resources"])):
        report.problem(f"resource {name}", "is in CREATE_PATH_PARAMS but isn't generated any more. Fix: remove it from CREATE_PATH_PARAMS")

    fixed = 0
    for name, entry in config["resources"].items():
        if name not in generated:
            continue
        create_path, create_method = entry["create"]["path"], entry["create"]["method"].lower()
        read_path, read_method = entry["read"]["path"], entry["read"]["method"].lower()
        params = path_params(create_path)
        known = CREATE_PATH_PARAMS.get(name, [])
        stale = [param for param in known if param not in params]
        if stale:
            report.problem(f"resource {name}", f"CREATE_PATH_PARAMS lists {', '.join(stale)}, which {create_path} doesn't have. Fix: set {json.dumps(name)}: {json.dumps(params)} in CREATE_PATH_PARAMS")

        create_params = spec.parameters(create_path, create_method)
        read_params = {param for location, param in spec.parameters(read_path, read_method) if location in ("path", "query")}
        aliases = entry.get("schema", {}).get("attributes", {}).get("aliases", {})
        attributes = generated[name]["schema"]["attributes"]
        unfixed = []
        for param in params:
            # Aliases rename read parameters, so a create parameter is renamed only if it's one.
            attribute_name = tf_identifier(aliases.get(param, param) if param in read_params else param)
            attribute = next((a for a in attributes if a["name"] == attribute_name), None)
            if param not in known:
                unfixed.append(f"{param} ({describe(attribute)})")
                continue
            if attribute is None:
                attribute = path_attribute(spec, attribute_name, create_params.get(("path", param)) or {})
                attributes.append(attribute)
            type_key = attribute_type(attribute)
            if type_key not in PLAN_MODIFIER_TYPES:
                report.problem(f"resource {name}", f"create path parameter {param} merged into a {type_key} attribute, which the fix doesn't handle")
                continue
            body = attribute[type_key]
            body["computed_optional_required"] = "required"
            body.pop("default", None)  # only computed attributes can have one
            body["plan_modifiers"] = [*body.get("plan_modifiers", []), requires_replace(type_key)]
            fixed += 1
        if unfixed:
            report.problem(
                f"resource {name}",
                f"create path parameters should be required and force replacement, but these aren't: {', '.join(unfixed)}. "
                f"Fix: set {json.dumps(name)}: {json.dumps(params)} in CREATE_PATH_PARAMS",
            )
    return fixed


def first_paragraph(text: str) -> str:
    """The first paragraph of an operation's description, unwrapped."""
    for paragraph in re.split(r"\n\s*\n", text.strip()):
        if paragraph.strip() and not paragraph.lstrip().startswith("**Required scope"):
            return " ".join(line.strip() for line in paragraph.splitlines())
    return ""


def required_scopes(op: dict) -> list[str]:
    return re.findall(r"\*\*Required scopes?:\*\*\s*`([^`]+)`", op.get("description") or "")


def add_descriptions(spec: Spec, config: dict, code_spec: dict) -> None:
    """Describe each resource and data source from its operations, for the documentation: what it
    is, and the scopes a token needs to manage or read it."""
    for config_key, spec_key, main in (("resources", "resources", "create"), ("data_sources", "datasources", "read")):
        for entry in code_spec.get(spec_key, []):
            locations = config[config_key][entry["name"]]
            ops = {key: spec.op(locations[key]["path"], locations[key]["method"].lower()) for key in ("create", "read", "update", "delete") if key in locations}
            op = ops[main]
            text = first_paragraph(op.get("description") or "")
            if config_key == "data_sources" and op.get("summary"):
                summary = " ".join(op["summary"].split()).rstrip(".") + "."
                text = f"{summary} {text}".strip()
            scopes = list(dict.fromkeys(scope for found in ops.values() for scope in required_scopes(found or {})))
            markdown = text
            if scopes:
                listed = [f"`{scope}`" for scope in scopes]
                markdown += "\n\nThe token needs " + (", ".join(listed[:-1]) + " and " if len(listed) > 1 else "") + listed[-1] + "."
            if markdown:
                entry["schema"]["description"] = re.sub(r"[`*]", "", markdown).replace("\n\n", " ")
                entry["schema"]["markdown_description"] = markdown


# String attributes that hold a credential, wherever they appear. The spec marks only some of them
# (format: password); Terraform shows the rest in plans and outputs unless they're sensitive.
SECRET_NAMES = {"secret", "token", "client_secret", "password", "private_key", "refresh_token", "api_token"}


def mark_secrets(code_spec: dict) -> int:
    def visit(attributes: list[dict]) -> int:
        marked = 0
        for attribute in attributes:
            body = attribute[attribute_type(attribute)]
            if attribute_type(attribute) == "string" and attribute["name"] in SECRET_NAMES and not body.get("sensitive"):
                body["sensitive"] = True
                marked += 1
            marked += visit(body.get("attributes") or body.get("nested_object", {}).get("attributes", []))
        return marked

    return sum(visit(entry["schema"]["attributes"]) for key in ("resources", "datasources") for entry in code_spec.get(key, []))


def fix_api_filled_nested(code_spec: dict) -> int:
    """Make the required attributes inside optional+computed nested attributes optional+computed.

    When the API fills in such an attribute the configuration left out, like an issue settings'
    statuses, its required inner attributes hold values. Terraform then takes those as having
    come from the configuration (optionalValueNotComputable), plans the attribute's removal, and
    never settles. The API still refuses a nested object without them."""

    def relax(attributes: list[dict]) -> int:
        changed = 0
        for attribute in attributes:
            body = attribute[attribute_type(attribute)]
            if body.get("computed_optional_required") == "required":
                body["computed_optional_required"] = "computed_optional"
                changed += 1
            changed += relax(body.get("attributes") or body.get("nested_object", {}).get("attributes", []))
        return changed

    def visit(attributes: list[dict]) -> int:
        changed = 0
        for attribute in attributes:
            body = attribute[attribute_type(attribute)]
            inner = body.get("attributes") or body.get("nested_object", {}).get("attributes", [])
            if body.get("computed_optional_required") == "computed_optional":
                changed += relax(inner)
            else:
                changed += visit(inner)
        return changed

    return sum(visit(resource["schema"]["attributes"]) for resource in code_spec.get("resources", []))


def path_attribute(spec: Spec, name: str, param: dict) -> dict:
    """A resource attribute for a path parameter the generator didn't map."""
    try:
        kind = ATTRIBUTE_TYPES.get(oas_type(spec.effective(param.get("schema") or {})), "string")
    except UnsupportedSchema:
        kind = "string"
    body = {"computed_optional_required": "required"}
    if param.get("description"):
        body["description"] = param["description"]
    return {"name": name, kind: body}


# Attribute types tfplugingen-framework generates a Go type and value for, named after the attribute.
NESTED_TYPES = {"single_nested", "list_nested", "set_nested", "map_nested"}


def go_name(name: str) -> str:
    """The prefix of the Go types generated for an attribute: FrameworkIdentifier.ToPascalCase."""
    return re.sub(r"^[a-z]|_[a-z0-9]", lambda match: match.group(0).replace("_", "").upper(), name)


def nested_attributes(attributes: list[dict], parent: tuple[str, ...] = ()) -> Iterator[tuple[tuple[str, ...], dict, list[dict]]]:
    """Every nested attribute under `attributes`, as (path, attribute, its siblings)."""
    for attribute in attributes:
        type_key = attribute_type(attribute)
        if type_key not in NESTED_TYPES:
            continue
        path = (*parent, attribute["name"])
        yield path, attribute, attributes
        body = attribute[type_key]
        yield from nested_attributes(body.get("attributes") or body.get("nested_object", {}).get("attributes", []), path)


def suggest_name(path: tuple[str, ...], siblings: list[dict], taken: set[str]) -> str:
    """The attribute's name prefixed with as few of its parents as it takes to be unique."""
    sibling_names = {sibling["name"] for sibling in siblings}
    candidates = ["_".join(path[start:]) for start in range(len(path) - 2, -1, -1)]
    candidates += [f"{path[-1]}_{number}" for number in range(2, 100)]
    return next(name for name in candidates if go_name(name) not in taken and name not in sibling_names)


def fix_nested_names(code_spec: dict, report: Report) -> tuple[int, dict[str, dict[str, str]]]:
    """Rename the nested attributes in RENAME_NESTED. Returns how many, and for each resource or data
    source, the JSON field behind each renamed attribute, by its path after renaming."""
    json_fields: dict[str, dict[str, str]] = {}
    schemas = {
        f"{kind} {entry['name']}": entry["schema"]["attributes"]
        for kind, key in (("resource", "resources"), ("data source", "datasources"))
        for entry in code_spec.get(key, [])
    }
    for what in sorted(set(RENAME_NESTED) - set(schemas)):
        report.problem(what, "is in RENAME_NESTED but isn't generated any more. Fix: remove it from RENAME_NESTED")

    renamed = 0
    for what, attributes in schemas.items():
        found = list(nested_attributes(attributes))  # before renaming, so paths use the generated names
        by_path = {".".join(path): (attribute, siblings) for path, attribute, siblings in found}
        counts = defaultdict(int)
        for _, attribute, _ in found:
            counts[go_name(attribute["name"])] += 1

        kept, originals = {}, {}
        for path, new_name in RENAME_NESTED.get(what, {}).items():
            if path not in by_path:
                report.problem(what, f"RENAME_NESTED renames {path}, which isn't a nested attribute any more. Fix: remove it from RENAME_NESTED")
                continue
            attribute, siblings = by_path[path]
            if counts[go_name(attribute["name"])] < 2:
                report.problem(what, f"RENAME_NESTED renames {path}, whose name doesn't clash any more. Fix: remove it from RENAME_NESTED")
                continue
            if any(sibling["name"] == new_name for sibling in siblings):
                report.problem(what, f"RENAME_NESTED renames {path} to {new_name}, which is already the name of an attribute next to it")
                continue
            originals[id(attribute)] = attribute["name"]
            attribute["name"] = new_name
            kept[path] = new_name
            renamed += 1
        if originals:
            json_fields[what] = {
                ".".join(path): originals[id(attribute)] for path, attribute, _ in nested_attributes(attributes) if id(attribute) in originals
            }

        groups = defaultdict(list)
        for order, (path, attribute, siblings) in enumerate(found):
            groups[go_name(attribute["name"])].append((len(path), order, path, siblings))
        clashes = [sorted(group, key=lambda item: item[:2]) for group in groups.values() if len(group) > 1]
        if not clashes:
            continue
        suggestion, taken = dict(kept), set(groups)
        for group in clashes:
            for _, _, path, siblings in group[1:]:
                name = suggest_name(path, siblings, taken)
                taken.add(go_name(name))
                suggestion[".".join(path)] = name
        listed = "; ".join(" and ".join(".".join(path) for _, _, path, _ in group) for group in clashes)
        report.problem(
            what,
            f"nested attributes with the same name generate the same Go types, which doesn't compile: {listed}. "
            f"Fix: set {json.dumps(what)}: {json.dumps(dict(sorted(suggestion.items())))} in RENAME_NESTED",
        )
    return renamed, json_fields


# --------------------------------------------------------------------------------------------------
# What the provider needs to call the API: provider_api.json
# --------------------------------------------------------------------------------------------------


def has_attribute_path(attributes: list[dict], dotted: str) -> bool:
    for part in dotted.split("."):
        attribute = next((a for a in attributes if a["name"] == part), None)
        if attribute is None:
            return False
        body = attribute[attribute_type(attribute)]
        attributes = body.get("attributes") or body.get("nested_object", {}).get("attributes", [])
    return True


def param_attribute(param: str, names: set[str], aliases: dict[str, str], entry_name: str) -> str | None:
    """The root attribute a parameter fills, found by name."""
    candidates = [tf_identifier(aliases[param])] if param in aliases else []
    candidates.append(tf_identifier(param))
    if "_" in param:
        stem, suffix = param.rsplit("_", 1)
        candidates.append(f"{ABBREVIATIONS.get(stem, stem)}_{suffix}")
        if entry_name.endswith((stem, ABBREVIATIONS.get(stem, stem))):
            candidates.append(suffix)  # the resource's own field: label_key -> key on a label
    return next((candidate for candidate in candidates if candidate in names), None)


def map_params(params: list[str], what: str, entry: dict, attributes: list[dict], report: Report) -> dict[str, str]:
    names = {attribute["name"] for attribute in attributes}
    aliases = entry.get("schema", {}).get("attributes", {}).get("aliases", {})
    known = PATH_PARAM_ATTRIBUTES.get(what, {})
    for param in sorted(set(known) - set(params)):
        report.problem(what, f"PATH_PARAM_ATTRIBUTES maps {param}, which no operation has any more. Fix: remove it from PATH_PARAM_ATTRIBUTES")
    mapped, unresolved = {}, []
    for param in params:
        found = param_attribute(param, names, aliases, what.split(" ")[-1])
        if param in known:
            if found:
                report.problem(what, f"PATH_PARAM_ATTRIBUTES maps {param}, which matches {found} by name now. Fix: remove it from PATH_PARAM_ATTRIBUTES")
            elif not has_attribute_path(attributes, known[param]):
                report.problem(what, f"PATH_PARAM_ATTRIBUTES maps {param} to {known[param]}, which isn't an attribute any more")
            mapped[param] = known[param]
        elif found:
            mapped[param] = found
        else:
            unresolved.append(param)
    if unresolved:
        fix = {param: "<attribute path>" for param in unresolved}
        report.problem(what, f"no attribute matches parameters {', '.join(unresolved)} by name. Fix: set {json.dumps(what)}: {json.dumps(fix)} in PATH_PARAM_ATTRIBUTES")
    return mapped


def body_fields(spec: Spec, location: dict, names: set[str]) -> dict[str, str]:
    """The request body fields an operation takes, as attribute -> JSON field."""
    op = spec.op(location["path"], location["method"].lower())
    request = spec.request_schema(op) if op else None
    properties = spec.root_properties(request) if request else {}
    return {tf_identifier(name): name for name in properties if tf_identifier(name) in names}


def paging(spec: Spec, location: dict, what: str, report: Report) -> dict | None:
    """How to read every page of a list, if it has pages."""
    path, method = location["path"], location["method"].lower()
    query = {name: param for (where, name), param in spec.parameters(path, method).items() if where == "query"}
    token = next((name for name in ("next_token", "cursor") if name in query), None)
    response = spec.response_schema(spec.op(path, method))
    try:
        properties = (spec.effective(response).get("properties") or {}) if response else {}
    except UnsupportedSchema:
        return None
    if token is None or "next_token" not in properties:
        return None
    arrays = []
    for name, prop in properties.items():
        try:
            if oas_type(spec.effective(prop)) == "array":
                arrays.append(name)
        except UnsupportedSchema:
            continue
    if len(arrays) != 1:
        report.warn(what, f"pages through {', '.join(arrays)} at once, so the provider only reads the first page")
        return None
    pages = {"param": token, "items": arrays[0]}
    try:
        maximum = spec.effective(query["limit"].get("schema") or {}).get("maximum") if "limit" in query else None
    except UnsupportedSchema:
        maximum = None
    if isinstance(maximum, int):
        pages["limit"] = maximum
    return pages


def list_field(spec: Spec, location: dict, what: str, report: Report) -> str | None:
    """The field holding the items of a list response, or None when the response is the list."""
    response = spec.response_schema(spec.op(location["path"], location["method"].lower()))
    try:
        schema = spec.effective(response) if response else {}
        if oas_type(schema) == "array":
            return None
    except UnsupportedSchema:
        return None
    arrays = []
    for name, prop in (schema.get("properties") or {}).items():
        try:
            if oas_type(spec.effective(prop)) == "array":
                arrays.append(name)
        except UnsupportedSchema:
            continue
    if len(arrays) != 1:
        report.problem(what, f"reads from a list, but its response has {len(arrays)} arrays: {', '.join(arrays) or 'none'}")
        return None
    return arrays[0]


def operation_entry(location: dict, **extra) -> dict:
    return {"method": location["method"], "path": location["path"], **{key: value for key, value in extra.items() if value}}


def resource_api(spec: Spec, name: str, entry: dict, attributes: list[dict], renames: dict, report: Report) -> dict:
    what = f"resource {name}"
    names = {attribute["name"] for attribute in attributes}
    locations = {key: entry[key] for key in ("create", "read", "update", "delete") if key in entry}
    all_params = list(dict.fromkeys(param for location in locations.values() for param in path_params(location["path"])))
    params = map_params(all_params, what, entry, attributes, report)

    create_body = body_fields(spec, entry["create"], names)
    api = {"create": operation_entry(entry["create"], body=create_body), "read": operation_entry(entry["read"])}
    update_body = {}
    if "update" in entry:
        update_body = body_fields(spec, entry["update"], names)
        api["update"] = operation_entry(entry["update"], body=update_body)
    if "delete" in entry:
        api["delete"] = operation_entry(entry["delete"])
    api["params"] = params

    # The object's own path identifies it: what a list read looks for. Import also takes the create
    # path's parameters, since an object created under its parent can live at a path without it
    # (POST /groups/{group_id}/teams, GET /teams/{team_id}), and they force replacement.
    item_path = (entry.get("delete") or entry.get("update") or entry["read"])["path"]
    read_path = entry["read"]["path"]
    identity = path_params(entry["create"]["path"]) + path_params(item_path)
    api["identity"] = list(dict.fromkeys(params[param] for param in identity if param in params))
    if item_path.startswith(read_path + "/"):
        # A list read: the item whose fields equal the object's identity is the object. Fields are
        # JSON paths, in the item as the Response override leaves it.
        own = [param for param in path_params(item_path) if param not in path_params(read_path) and param in params]
        api["match"] = {params[param]: params[param] for param in own}
        api["read"] = operation_entry(entry["read"], items=list_field(spec, entry["read"], what, report), pages=paging(spec, entry["read"], what, report))

    # What only create can set has to force replacement. Create path parameters already do.
    create_params = {params[param] for param in path_params(entry["create"]["path"]) if param in params}
    replace = set(create_body) - set(update_body) - create_params
    if replace:
        api["replace"] = sorted(replace)
    if renames:
        api["renames"] = renames
    return api


def data_source_api(spec: Spec, name: str, entry: dict, attributes: list[dict], renames: dict, report: Report) -> dict:
    what = f"data source {name}"
    names = {attribute["name"] for attribute in attributes}
    location = entry["read"]
    path, method = location["path"], location["method"].lower()
    aliases = entry.get("schema", {}).get("attributes", {}).get("aliases", {})
    query = {}
    for where, param in spec.parameters(path, method):
        attribute = param_attribute(param, names, aliases, name) if where == "query" and param not in PAGINATION_FIELDS else None
        if attribute:
            query[param] = attribute
    api = {
        "read": operation_entry(location, query=query, pages=paging(spec, location, what, report)),
        "params": map_params(path_params(path), what, entry, attributes, report),
    }
    try:
        is_array = oas_type(spec.effective(spec.response_schema(spec.op(path, method)))) == "array"
    except UnsupportedSchema:
        is_array = False
    if is_array:
        # The generator puts an array response in an attribute named after the data source.
        if tf_identifier(name) in names:
            api["root"] = tf_identifier(name)
        else:
            report.problem(what, f"its response is an array, but it has no {tf_identifier(name)} attribute to hold it")
    if renames:
        api["renames"] = renames
    return api


def build_api_map(spec: Spec, config: dict, code_spec: dict, renames: dict[str, dict[str, str]], report: Report) -> dict:
    api_map = {"resources": {}, "data_sources": {}}
    for kind, config_key, spec_key, build in (
        ("resource", "resources", "resources", resource_api),
        ("data source", "data_sources", "datasources", data_source_api),
    ):
        generated = {entry["name"]: entry["schema"]["attributes"] for entry in code_spec.get(spec_key, [])}
        for name, entry in config[config_key].items():
            if name in generated:
                api_map[config_key][name] = build(spec, name, entry, generated[name], renames.get(f"{kind} {name}", {}), report)
    for what in sorted(PATH_PARAM_ATTRIBUTES):
        kind, _, name = what.rpartition(" ")
        if name not in api_map["resources" if kind == "resource" else "data_sources"]:
            report.problem(what, "is in PATH_PARAM_ATTRIBUTES but isn't generated any more. Fix: remove it from PATH_PARAM_ATTRIBUTES")
    return api_map


# --------------------------------------------------------------------------------------------------
# Running the generator, and output
# --------------------------------------------------------------------------------------------------


class ConfigDumper(yaml.SafeDumper):
    def ignore_aliases(self, data):
        return True


def deep_merge(base: dict, overrides: dict) -> dict:
    for key, value in overrides.items():
        if value is None:
            base.pop(key, None)
        elif isinstance(value, dict) and isinstance(base.get(key), dict):
            deep_merge(base[key], value)
        elif isinstance(value, list) and isinstance(base.get(key), list):
            base[key] = sorted(set(base[key]) | set(value))
        else:
            base[key] = value
    return base


def run_generator(config_path: Path, doc: dict, report: Report) -> dict | None:
    # The version is pinned by the tool directive in go.mod: the mapping rules above mirror it.
    if not shutil.which("go"):
        print("error: go is not on PATH; it runs tfplugingen-openapi at the version pinned in go.mod", file=sys.stderr)
        return None
    with tempfile.TemporaryDirectory() as directory:
        spec_path = Path(directory) / "openapi.json"
        spec_path.write_text(json.dumps(doc))
        output = Path(directory) / "provider_code_spec.json"
        result = subprocess.run(
            ["go", "-C", str(REPO_ROOT), "tool", "tfplugingen-openapi", "generate", "--config", str(config_path.resolve()), "--output", str(output), str(spec_path)],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0 or not output.exists():
            print(f"error: tfplugingen-openapi failed\n{result.stdout}\n{result.stderr}", file=sys.stderr)
            return None
        code_spec = json.loads(output.read_text())
    for line in result.stderr.splitlines():
        if "WARN" in line and "skipping" in line:
            report.problem("generator", f"didn't map something the script expected it to: {line}")
    return code_spec


def check_output(config: dict, code_spec: dict, report: Report) -> None:
    for kind, config_key, spec_key in (("resource", "resources", "resources"), ("data source", "data_sources", "datasources")):
        generated = {entry["name"] for entry in code_spec.get(spec_key, [])}
        for name in sorted(set(config.get(config_key, {})) - generated):
            report.problem(f"{kind} {name}", "is in the config, but the generator left it out")


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate generator_config.yml and provider_code_spec.json from an OpenAPI spec.")
    parser.add_argument("--spec", type=Path, default=Path("openapi.json"), help="OpenAPI spec, JSON or YAML (default: %(default)s)")
    parser.add_argument("--output", type=Path, default=Path("generator_config.yml"), help="generator config to write (default: %(default)s)")
    parser.add_argument("--code-spec", type=Path, default=Path("provider_code_spec.json"), help="provider code spec to write (default: %(default)s)")
    parser.add_argument("--api-map", type=Path, default=Path("provider_api.json"), help="how the provider calls the API, for scripts/generate_provider.py (default: %(default)s)")
    parser.add_argument("--overrides", type=Path, default=Path("generator_overrides.yml"), help="manual changes merged over the config, if the file exists (default: %(default)s)")
    parser.add_argument("-v", "--verbose", action="store_true", help="list every skip and warning")
    args = parser.parse_args()

    # yaml.safe_load also reads JSON, but is far slower on a spec this size.
    text = args.spec.read_text()
    doc = json.loads(text) if args.spec.suffix == ".json" else yaml.safe_load(text)
    spec = Spec(doc)
    report = Report()
    merged = merge_known_allof(spec, report)

    resources: dict[str, dict] = {}
    for shape in discover_resources(spec, report):
        if shape.name in resources:
            print(f"error: two resources would be named {shape.name}, at {shape.read[0]} and {resources[shape.name]['read']['path']}", file=sys.stderr)
            return 1
        entry = build_resource(spec, shape, report)
        if entry:
            resources[shape.name] = entry

    config = {
        "provider": {"name": PROVIDER_NAME},
        "resources": dict(sorted(resources.items())),
        "data_sources": dict(sorted(build_data_sources(spec, report).items())),
    }
    if args.overrides.exists():
        deep_merge(config, yaml.safe_load(args.overrides.read_text()) or {})

    info = doc.get("info", {})
    header = (
        f"# Generated by scripts/generate_generator_config.py from {args.spec.name} "
        f"({info.get('title')} {info.get('version')}).\n"
        f"# Don't edit by hand, it gets overwritten. Manual changes go in {args.overrides.name}.\n"
        f"# The script runs tfplugingen-openapi on a corrected copy of the spec and corrects its output,\n"
        f"# so run the script rather than the generator.\n"
    )
    args.output.write_text(header + yaml.dump(config, Dumper=ConfigDumper, sort_keys=False, width=1000))

    code_spec = run_generator(args.output, doc, report)
    if code_spec is None:
        return 1
    check_output(config, code_spec, report)
    fixed = fix_create_path_params(spec, config, code_spec, report)
    relaxed = fix_api_filled_nested(code_spec)
    secrets = mark_secrets(code_spec)
    add_descriptions(spec, config, code_spec)
    renamed, renamed_fields = fix_nested_names(code_spec, report)
    api_map = build_api_map(spec, config, code_spec, renamed_fields, report)
    report_unmerged_allof(spec, report)
    args.code_spec.write_text(json.dumps(code_spec, indent="\t") + "\n")
    args.api_map.write_text(json.dumps(api_map, indent="\t", sort_keys=True) + "\n")

    print(f"wrote {args.output}: {len(config['resources'])} resources, {len(config['data_sources'])} data sources", file=sys.stderr)
    print(
        f"wrote {args.code_spec}: merged {merged} allOf schemas, made {fixed} create path parameters required and force replacement, "
        f"made {relaxed} attributes the API fills in optional, marked {secrets} secrets sensitive, renamed {renamed} nested attributes",
        file=sys.stderr,
    )
    print(f"wrote {args.api_map}: the operations behind each resource and data source", file=sys.stderr)
    print(f"{len(report.skipped)} skipped, {len(report.warnings)} warnings" + ("" if args.verbose else " (-v to list them)"), file=sys.stderr)
    if args.verbose:
        for what, reason in report.skipped:
            print(f"  skipped  {what}: {reason}", file=sys.stderr)
        for what, message in report.warnings:
            print(f"  warning  {what}: {message}", file=sys.stderr)
    if report.problems:
        print(f"{len(report.problems)} problems to fix:", file=sys.stderr)
        for what, message in report.problems:
            print(f"  {what}: {message}", file=sys.stderr)
        return 1
    print("no problems", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
