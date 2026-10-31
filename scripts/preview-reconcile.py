#!/usr/bin/env python3
"""Turn a PR environment's variable listing into the variables to write.

usage: preview-reconcile.py --manifest FILE --pr NUMBER --frontend-host HOST
                            [LISTING.json]

LISTING is the JSON object of `railway variable list --json` (default: stdin).
Prints {"variables": {...}, "removals": [...], "secrets": {...}}: variables is
the complete replacement set (the listing without RAILWAY_* and the removals,
plus the overrides and derived values), removals the remove-strategy variables
the listing has, and secrets maps each variable whose override is a secret
(`secret: true` in the manifest) to its source. That variable holds
@@SOURCE@@ in place of the value: this script never receives a secret, the
workflow fills it in afterwards.
Exits 1, naming the variable, on one the manifest does not classify, on a
sealed one it cannot carry over, or when the derived APP_CLIENT_URLS is not
https://HOST.
"""
import argparse
import json
import os
import sys

import yaml

PLATFORM_PREFIX = "RAILWAY_"


def plan(manifest, listing, pr, env, frontend_host):
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
    secrets = {}
    for name, spec in manifest.get("override", {}).items():
        if "value" in spec:
            wanted[name] = spec["value"]
        elif spec.get("secret"):
            wanted[name] = f"@@{spec['source']}@@"
            secrets[name] = spec["source"]
        elif spec["source"] in env:
            wanted[name] = env[spec["source"]]
        else:
            raise ValueError(f"{name}: environment variable {spec['source']} is not set")
    for name, template in manifest.get("derive", {}).items():
        wanted[name] = template.replace("{pr}", str(pr))

    removals = sorted(n for n in groups["remove"] if n in listing)
    # A sealed variable lists with a null value and cannot be read back, so
    # replacing the set would lose it unless it is removed or overwritten.
    sealed = sorted(
        n for n, v in listing.items()
        if v is None and not n.startswith(PLATFORM_PREFIX) and n not in removals and n not in wanted
    )
    if sealed:
        raise ValueError(f"sealed variables would be lost by the update and are not in the manifest's remove list: {', '.join(sealed)}")

    variables = {n: v for n, v in listing.items() if not n.startswith(PLATFORM_PREFIX) and n not in removals}
    variables.update(wanted)
    if variables.get("APP_CLIENT_URLS") != f"https://{frontend_host}":
        raise ValueError(f"the derived APP_CLIENT_URLS is {variables.get('APP_CLIENT_URLS')!r}, not https://{frontend_host}, the address the workflow aliases")
    return {"variables": variables, "removals": removals, "secrets": secrets}


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--manifest", required=True)
    p.add_argument("--pr", required=True, type=int)
    p.add_argument("--frontend-host", required=True)
    p.add_argument("listing", nargs="?", default="-")
    args = p.parse_args()
    with open(args.manifest) as f:
        manifest = yaml.safe_load(f)
    listing = json.load(sys.stdin if args.listing == "-" else open(args.listing))
    try:
        result = plan(manifest, listing, args.pr, os.environ, args.frontend_host)
    except ValueError as e:
        sys.exit(f"preview-reconcile: {e}")
    json.dump(result, sys.stdout, indent=2, sort_keys=True)
    print()


if __name__ == "__main__":
    main()
