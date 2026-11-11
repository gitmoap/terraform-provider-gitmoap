"""Rewrite what Terraform has no type for into what it has, in the copy of the spec the generator reads.

Terraform has no union type, and tfplugingen-openapi maps a oneOf or anyOf only when it's a type
and null, or a string and a type that converts to one: anything else, it drops with the whole body
around it. So, before it runs:

- A union of one type, besides null, becomes that type.
- A union of objects becomes one object with every variant's fields, all optional: you set those
  of the variant you mean. Both ways an enum is written in JSON map like this, without changing
  the JSON: tagged by a field, {"kind": "range", "min": 1}, whose tags become one enum; and tagged
  by the variant's name, {"person": "…"}, where a variant holding an empty object, {"anyone": {}},
  becomes a bool, anyone = true.
- A union of strings and an object becomes the object, with a value field for the strings:
  "disabled" is value = "disabled", and {"to": "done"} is to = "done".
- A free-form object, or a union none of the above fits, becomes a string of JSON, set with
  jsonencode().

A rewritten schema carries MARK, saying how the provider translates its values: marker (a bool
for an empty object), scalar_or_object (a value field for the strings), or json. The JSON of
everything else doesn't change, so it needs no translation.
"""

from __future__ import annotations

import json
from collections import Counter

MARK = "x-gitmoap-transform"
JSON_NOTE = "\n\nJSON: set it with `jsonencode()`."
WAS = "x-gitmoap-was"  # what a JSON string was before: an empty object can still become a marker
SCALARS = {"string", "integer", "number", "boolean"}
ANNOTATIONS = ("description", "title", "examples", "example", "default", "format", "readOnly", "writeOnly", "deprecated")


def schemas_in(doc: dict):
    """Every schema root of the document: components, parameters, bodies."""
    components = doc.get("components", {})
    yield from components.get("schemas", {}).values()
    containers = list(components.get("parameters", {}).values())
    containers += list(components.get("requestBodies", {}).values()) + list(components.get("responses", {}).values())
    for item in doc.get("paths", {}).values():
        containers += [parameter for parameter in item.get("parameters", [])]
        for operation in item.values():
            if not isinstance(operation, dict):
                continue
            containers += operation.get("parameters", [])
            containers.append(operation.get("requestBody", {}))
            containers += list(operation.get("responses", {}).values())
    for container in containers:
        if not isinstance(container, dict):
            continue
        if isinstance(container.get("schema"), dict):
            yield container["schema"]
        for media in (container.get("content") or {}).values():
            if isinstance(media.get("schema"), dict):
                yield media["schema"]


def visit(node, fn, seen: set[int]) -> None:
    """Call fn on every schema in node, inner schemas first."""
    if not isinstance(node, dict) or id(node) in seen:
        return
    seen.add(id(node))
    for child in (node.get("properties") or {}).values():
        visit(child, fn, seen)
    for key in ("items", "additionalProperties", "not"):
        if isinstance(node.get(key), dict):
            visit(node[key], fn, seen)
    for key in ("oneOf", "anyOf", "allOf"):
        for child in node.get(key) or []:
            visit(child, fn, seen)
    fn(node)


class Rewriter:
    def __init__(self, spec):
        self.spec = spec
        self.counts: Counter = Counter()
        self.fallbacks: list[str] = []

    def deref(self, node) -> dict:
        return self.spec.deref(node)

    def kind(self, node) -> str:
        schema = self.deref(node)
        if schema.get("oneOf") or schema.get("anyOf"):
            return "union"
        if MARK in schema:
            return {"json": "string", "marker": "boolean"}.get(schema[MARK], "object")
        if schema.get("allOf") or schema.get("properties") or schema.get("type") == "object":
            return "object"
        kind = schema.get("type")
        if isinstance(kind, list):
            kinds = [k for k in kind if k != "null"]
            kind = kinds[0] if len(kinds) == 1 else "mixed" if kinds else "null"
        if kind is None and "const" in schema:
            kind = "string" if isinstance(schema["const"], str) else "number"
        if kind is None and schema.get("enum"):
            kind = "string"
        return kind or "any"

    def parts(self, node) -> tuple[dict, list]:
        """The properties and required names of an object variant."""
        schema = self.deref(node)
        if schema.get("allOf"):
            return self.spec.allof_members(schema)
        return dict(schema.get("properties") or {}), list(schema.get("required") or [])

    @staticmethod
    def is_free_form(schema: dict) -> bool:
        if "$ref" in schema or MARK in schema or schema.get("properties") or schema.get("enum") or "const" in schema:
            return False
        if any(schema.get(key) for key in ("oneOf", "anyOf", "allOf", "items")) or isinstance(schema.get("additionalProperties"), dict) and schema["additionalProperties"]:
            return False
        kind = schema.get("type")
        return kind == "object" or (isinstance(kind, list) and "object" in kind)

    def tags(self, node) -> list | None:
        schema = self.deref(node)
        if "const" in schema and isinstance(schema["const"], str):
            return [schema["const"]]
        if schema.get("enum") and all(isinstance(value, str) for value in schema["enum"]):
            return list(schema["enum"])
        return None

    def normalized(self, node) -> str:
        schema = {key: value for key, value in self.deref(node).items() if key not in ANNOTATIONS}
        kind = schema.get("type")
        if isinstance(kind, list):
            kinds = [k for k in kind if k != "null"]
            schema["type"] = kinds[0] if len(kinds) == 1 else kinds
        return json.dumps(schema, sort_keys=True)

    def combine(self, first, second):
        """One schema for a field two variants both have, or None if they disagree."""
        if self.normalized(first) == self.normalized(second):
            return first
        tags_first, tags_second = self.tags(first), self.tags(second)
        if tags_first is not None and tags_second is not None:
            return {"type": "string", "enum": sorted(set(tags_first) | set(tags_second))}
        if self.kind(first) == "object" and self.kind(second) == "object":
            merged = self.merge([first, second])
            return merged[0] if merged else None
        return None

    def merge(self, variants: list) -> tuple[dict, str] | None:
        """One object for object variants, and how to tell the variants apart, or None."""
        properties: dict = {}
        owners: list[tuple[list, list]] = []
        for variant in variants:
            props, required = self.parts(variant)
            owners.append((list(props), required))
            for name, prop in props.items():
                if name not in properties:
                    properties[name] = prop
                    continue
                combined = self.combine(properties[name], prop)
                if combined is None:
                    return None
                properties[name] = combined

        # Tagged by the variant's name: each variant one required field, all different.
        names = [props[0] for props, required in owners if len(props) == 1 and props == required]
        if len(names) == len(variants) and len(set(names)) == len(names):
            for name in names:
                value = self.deref(properties[name])
                if self.is_free_form(value) or value.get(WAS) == "free-form object":
                    described = (value.get("description") or "").split(JSON_NOTE)[0].strip()
                    properties[name] = {"type": "boolean", MARK: "marker", **({"description": described} if described else {})}
            listed = ", ".join(f"`{name}`" for name in names[:-1]) + f" or `{names[-1]}`"
            return {"type": "object", "properties": properties}, f"Set one of {listed}."

        # Tagged by a field every variant has, with a different value in each.
        for tag in properties:
            values = [self.tags(self.parts(variant)[0].get(tag, {})) for variant in variants]
            if all(value and len(value) == 1 for value in values) and len({value[0] for value in values}) == len(values):
                described = []
                for (props, _), value in zip(owners, values):
                    others = [f"`{name}`" for name in props if name != tag]
                    described.append(f"`{value[0]}`" + (f", with {', '.join(others)}" if others else ""))
                return {"type": "object", "properties": properties}, f"`{tag}` says which applies: {'; '.join(described)}."
        return {"type": "object", "properties": properties}, "Set the fields of one of its forms."

    def rewrite(self, node: dict) -> None:
        if self.is_free_form(node):
            self.counts["free-form object"] += 1
            self.to_json(node, was="free-form object")
            return
        variants = node.get("oneOf") or node.get("anyOf")
        if not variants:
            return
        kinds = [self.kind(variant) for variant in variants]
        real = [variant for variant, kind in zip(variants, kinds) if kind != "null"]
        real_kinds = [kind for kind in kinds if kind != "null"]
        nullable = len(real) < len(variants)
        if nullable and len(variants) == 2:
            return  # the generator maps a type or null
        if len(real) == 2 and not nullable and set(real_kinds) <= SCALARS and "string" in real_kinds:
            return  # and a string, or something that converts to one
        keep = {key: value for key, value in node.items() if key in ANNOTATIONS}
        if len(real) == 1:
            replacement, note, how = {"allOf": [real[0]]}, None, "one type"
        elif all(kind == "object" for kind in real_kinds):
            merged = self.merge(real)
            if merged is None:
                return self.fallback(node, "its forms disagree on a field")
            (replacement, note), how = merged, "objects"
        elif set(real_kinds) <= SCALARS:
            replacement, note, how = {"type": "string"}, None, "scalars"
        elif set(real_kinds) <= SCALARS | {"object"}:
            scalars = [variant for variant, kind in zip(real, real_kinds) if kind != "object"]
            objects = [variant for variant, kind in zip(real, real_kinds) if kind == "object"]
            merged = self.merge(objects) if len(objects) > 1 else ({"type": "object", "properties": self.parts(objects[0])[0]}, None)
            if merged is None or "value" in merged[0]["properties"]:
                return self.fallback(node, "its object forms disagree")
            replacement = merged[0]
            values = sorted({tag for variant in scalars for tag in (self.tags(variant) or [])})
            listed = ", ".join(f"`{tag}`" for tag in values) or "a string"
            value = {"type": "string", **({"enum": values} if values else {}), "description": f"One of {listed}, instead of the other fields."}
            replacement["properties"] = {"value": value, **replacement["properties"]}
            replacement[MARK] = "scalar_or_object"
            listed = ", ".join(f"`{tag}`" for tag in values) or "a string"
            note = f"Set `value` to {listed}, or set the other fields."
            how = "strings or an object"
        else:
            return self.fallback(node, f"it mixes {', '.join(sorted(set(real_kinds)))}")
        if note:
            keep["description"] = (keep.get("description", "").rstrip() + "\n\n" + note).strip()
        node.clear()
        if nullable and len(real) > 1:
            node.update({"oneOf": [{"type": "null"}, {**replacement, **keep}]})
        else:
            node.update({**replacement, **keep})
        self.counts[how] += 1

    def fallback(self, node: dict, why: str) -> None:
        self.fallbacks.append(why)
        self.counts["union as JSON"] += 1
        self.to_json(node)

    @staticmethod
    def to_json(node: dict, was: str = "union") -> None:
        description = (node.get("description") or "").rstrip()
        node.clear()
        node.update({"type": "string", MARK: "json", WAS: was, "description": (description + JSON_NOTE).strip()})


def rewrite_unions(spec) -> Counter:
    """Rewrite every union and free-form object in the spec, in place."""
    rewriter = Rewriter(spec)
    seen: set[int] = set()
    for root in list(schemas_in(spec.doc)):
        visit(root, rewriter.rewrite, seen)
    return rewriter.counts


def marks_in(spec, node, path: tuple = (), seen: frozenset = frozenset()) -> dict[tuple, str]:
    """Where a body holds a rewritten schema, by attribute path, the way the generator maps it."""
    from generate_generator_config import UnsupportedSchema, oas_type, tf_identifier

    try:
        schema = spec.effective(node)
        kind = oas_type(schema)
    except (UnsupportedSchema, KeyError):
        return {}
    found = {path: schema[MARK]} if MARK in schema and path else {}
    if schema.get(MARK) == "json" or id(schema) in seen:
        return found
    seen = seen | {id(schema)}
    if kind == "array" and isinstance(schema.get("items"), dict):
        found.update(marks_in(spec, schema["items"], path, seen))
    if isinstance(schema.get("additionalProperties"), dict):
        found.update(marks_in(spec, schema["additionalProperties"], path, seen))
    for name, prop in (schema.get("properties") or {}).items():
        found.update(marks_in(spec, prop, (*path, tf_identifier(name)), seen))
    return found
