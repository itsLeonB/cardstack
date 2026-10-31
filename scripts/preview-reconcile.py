#!/usr/bin/env python3
"""Turn a PR environment's variable listing into the changes to apply.

usage: preview-reconcile.py --manifest FILE --pr NUMBER [LISTING.json]

LISTING is the JSON object of `railway variable list --json` (default: stdin).
Prints {"sets": {...}, "removals": [...]}: overrides and derived values that
differ from the listing, and remove-strategy variables the listing has.
Exits 1, naming the variable, on one the manifest does not classify.
"""
import argparse
import json
import os
import sys

import yaml

PLATFORM_PREFIX = "RAILWAY_"


def plan(manifest, listing, pr, env):
    groups = {
        "inherit": set(manifest.get("inherit", [])),
        "override": set(manifest.get("override", {})),
        "derive": set(manifest.get("derive", {})),
        "remove": set(manifest.get("remove", [])),
    }
    names = sorted(n for g in groups.values() for n in g)
    dupes = sorted({n for n in names if names.count(n) > 1})
    if dupes:
        raise ValueError(f"classified under more than one strategy: {', '.join(dupes)}")
    known = set(names)
    unknown = sorted(n for n in listing if n not in known and not n.startswith(PLATFORM_PREFIX))
    if unknown:
        raise ValueError(f"not in the manifest: {', '.join(unknown)}")

    wanted = {}
    for name, spec in manifest.get("override", {}).items():
        if "value" in spec:
            wanted[name] = spec["value"]
        elif spec["source"] in env:
            wanted[name] = env[spec["source"]]
        else:
            raise ValueError(f"{name}: environment variable {spec['source']} is not set")
    for name, template in manifest.get("derive", {}).items():
        wanted[name] = template.replace("{pr}", str(pr))

    return {
        "sets": {n: v for n, v in sorted(wanted.items()) if listing.get(n) != v},
        "removals": sorted(n for n in groups["remove"] if n in listing),
    }


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--manifest", required=True)
    p.add_argument("--pr", required=True, type=int)
    p.add_argument("listing", nargs="?", default="-")
    args = p.parse_args()
    with open(args.manifest) as f:
        manifest = yaml.safe_load(f)
    listing = json.load(sys.stdin if args.listing == "-" else open(args.listing))
    try:
        result = plan(manifest, listing, args.pr, os.environ)
    except ValueError as e:
        sys.exit(f"preview-reconcile: {e}")
    json.dump(result, sys.stdout, indent=2, sort_keys=True)
    print()


if __name__ == "__main__":
    main()
