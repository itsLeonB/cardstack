# shellcheck shell=bash
# Shared by the verification scripts: the compose stack of this worktree.
# The project name is Compose's default, this directory's name, so each
# worktree has its own stack.

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOST_UID="$(id -u)"
HOST_GID="$(id -g)"
export HOST_UID HOST_GID

compose() {
  docker compose --project-directory "$root" "$@"
}

# up <service>...: starts these services (and what they depend on) and waits
# until they are healthy. --build is a no-op when an image is current. CI
# builds the images first, with a layer cache, and sets SKIP_BUILD.
up() {
  local build=--build
  [ -z "${SKIP_BUILD:-}" ] || build=--no-build
  compose up -d "$build" --wait "$@"
}

# run_in <service> <command>...: runs a command in a started tooling container.
run_in() {
  local service=$1
  shift
  echo "+ $*"
  compose exec -T "$service" "$@"
}
