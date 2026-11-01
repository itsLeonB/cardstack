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

# serialized <command>...: runs a CPU-bound command holding a host-wide lock,
# so suites from several worktrees queue instead of oversubscribing the cores:
# the timing-sensitive frontend tests time out when stacks run them at once.
# Without flock (macOS) it runs unlocked.
serialized() {
  (
    if command -v flock >/dev/null; then
      flock -n 9 || echo "waiting for another worktree's run to finish..."
      flock 9
    fi
    "$@"
  ) 9>"${TMPDIR:-/tmp}/cardstack-verification.lock"
}

# run_in <service> <command>...: runs a command in a started tooling container,
# in $RUN_DIR (relative to the repository root) when it is set, with each
# variable named in $RUN_ENV passed through from the shell when it is set.
run_in() {
  local service=$1 args=() var
  shift
  [ -z "${RUN_DIR:-}" ] || args+=(-w "/workspace/$RUN_DIR")
  for var in ${RUN_ENV:-}; do
    [ -z "${!var+x}" ] || args+=(-e "$var")
  done
  echo "+ ${RUN_DIR:+($RUN_DIR) }$*"
  # The ${args[@]+...} form keeps an empty array safe under `set -u` on Bash 3.2.
  compose exec -T ${args[@]+"${args[@]}"} "$service" "$@"
}
