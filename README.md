# todo-age

Lists TODOs with git-blame age as a table, JSON or CSV.

Every codebase collects `TODO` comments. Most are fine; some have been
sitting there for three years. `todo-age` finds every `TODO`, `FIXME`, `HACK`
and `XXX` in the tracked files of a repository, asks `git blame` who wrote
each line and when, and lists them oldest first. In CI it can fail the build
when something gets too old.

## Install

```sh
go install github.com/dan2756/todo-age@latest
```

Requires Go 1.24 or newer to build and `git` on your `PATH` to run.

## Usage

```
todo-age [flags] [path ...]
```

Run it anywhere inside a git work tree:

```sh
$ todo-age
AGE     DATE        AUTHOR             TAG         LOCATION              TEXT
1y 5mo  2025-04-01  Maria Lopez        TODO        server/handler.go:88  validate the redirect target
7mo     2026-02-10  Sam Okafor         FIXME(sam)  store/cache.go:14     eviction ignores TTL
5w      2026-08-12  Priya Raman        XXX         api/routes.go:203     drop after the v2 cutover
3d      2026-09-16  Maria Lopez        HACK        cmd/main.go:41        remove once v2 config ships
today   2026-09-19  Not Committed Yet  TODO        store/pool.go:57      retry on ECONNRESET
```

Only tracked files are scanned, skipping binary files, files over 1 MiB,
symlinks and submodules. Lines you have changed but not committed show up as
`Not Committed Yet` with an age of 0, as do all markers in a repository
without commits. Run from a subdirectory, it scans that subdirectory, and
paths are relative to where you are, as with `git ls-files`.

### Examples

```sh
todo-age src/ cmd/                      # only files under these paths
todo-age --tag FIXME                    # only FIXMEs
todo-age --tags TODO,FIXME,BUG          # look for a different set of markers
todo-age --tag BUG                      # a marker outside the default set
todo-age --exclude 'vendor/**,*.min.js' # skip generated and third-party code
todo-age --older-than 90d               # stale ones only
todo-age --newer-than 2w                # what came in recently
todo-age --author '(?i)maria'           # regex on author name or email
todo-age --sort author                  # or file; the default is oldest first
todo-age --limit 10                     # the ten oldest
todo-age --format csv > todos.csv       # for a spreadsheet
todo-age --format ndjson | jq -r 'select(.age_days > 365) | .email' | sort | uniq -c
```

### Flags

| Flag | Meaning |
|------|---------|
| `-C dir` | Run as if started in `dir`. |
| `--tags list` | Markers to look for. Default `TODO,FIXME,HACK,XXX`. |
| `--exclude list` | Skip files matching these globs (see below). |
| `--max-size size` | Skip files larger than this: `512K`, `4M`, `1MiB`. Default `1M`; `0` means no limit. |
| `--tag list` | Only show these markers (case-insensitive). Markers not in `--tags` are added to it in upper case, so `--tag bug` finds `BUG`. |
| `--author regex` | Only show lines whose author name or email matches. |
| `--older-than age` | Only show markers older than `age`. |
| `--newer-than age` | Only show markers newer than `age`. |
| `--sort key` | `age` (oldest first, the default), `file` (file, then line) or `author`. |
| `--limit n` | Show at most `n` markers. |
| `--format fmt` | `table` (default), `json`, `csv` or `ndjson`. |
| `--summary` | Print counts per tag and age bucket instead of the markers. |
| `--fail-older-than age` | Exit with status 1 if any marker is older than `age`. |
| `--version` | Print the version. |
| `-h`, `--help` | Show help. |

List flags take comma-separated values and may also be repeated:
`--exclude vendor --exclude '*.pb.go'`. Flags may come before or after paths.

Ages are one or more number-unit pairs: `h`, `d`, `w`, `mo` (30 days) and
`y` (365 days), so `90d`, `6mo`, `1y` and `1y6mo` all work. A bare number is
days.

Exclude patterns without a slash (`*.min.js`, `vendor`) match a file or
directory name at any depth. Patterns with a slash at the start or in the
middle (`/docs`, `vendor/**`, `web/*.css`) match from the top of the
scanned tree, and `**` spans any number of directories. As in
`.gitignore`, a trailing slash alone does not anchor a pattern: `docs/`
matches at any depth, like `docs`. A pattern that matches a directory skips
everything in it.

## What counts as a marker

A marker is one of the tags as a whole, case-sensitive word, optionally
followed by an owner in parentheses and a colon. The rest of the line is
its text:

| Line | Tag | Owner | Text |
|------|-----|-------|------|
| `// TODO: handle timeouts` | `TODO` | | `handle timeouts` |
| `# FIXME(sam): leaks on error` | `FIXME` | `sam` | `leaks on error` |
| `/* HACK until the v2 client ships */` | `HACK` | | `until the v2 client ships` |
| `<!-- XXX - remove before launch -->` | `XXX` | | `remove before launch` |

`TODOS`, `todo`, `TODO_LIST` and a tag that is itself a quoted string, such
as `[]string{"TODO"}`, are not markers. Only the first marker on a line is
reported.

## Output

Every row has these fields, in this order in CSV:

| Field | Example | Notes |
|-------|---------|-------|
| `file` | `server/handler.go` | |
| `line` | `88` | |
| `tag` | `TODO` | |
| `text` | `validate the redirect target` | |
| `owner` | `sam` | From `TODO(sam)`; empty otherwise. |
| `author` | `Maria Lopez` | `Not Committed Yet` for local changes. |
| `email` | `maria@example.com` | Empty for local changes. |
| `commit` | `3f9c2a1` | Short hash; empty for local changes. |
| `date` | `2025-04-01` | Author date, in the author's time zone. |
| `age_days` | `536` | Whole days since the author date. |
| `uncommitted` | `false` | |

`--format json` prints an array of these objects, `--format ndjson` one
object per line, and `--format csv` a header row followed by one record per
marker, quoted as RFC 4180 requires.

## Summary

`--summary` prints counts instead of the markers themselves. Filters apply;
`--limit` does not.

```sh
$ todo-age --summary
TAG      COUNT
TODO        41
FIXME       12
HACK         3
XXX          1

AGE      COUNT
<1w          2
1w-1mo       5
1mo-3mo      9
3mo-6mo      8
6mo-1y      14
1y-2y       11
>2y          8

TOTAL       57
```

With `--format json` it is an object with `total`, `by_tag` and `by_age`;
with `csv` and `ndjson` it is one `group,key,count` entry per line.

## In CI

`--fail-older-than` turns the listing into a check. It exits with status 1
if any marker that survives the filters is older than the given age, and
prints how many there were to stderr:

```sh
$ todo-age --fail-older-than 180d --exclude 'vendor/**'
...
todo-age: 3 markers older than 180d
$ echo $?
1
```

`--limit` only shortens the listing; the check still sees every marker.
Check out the full history in CI (for example `fetch-depth: 0` with
`actions/checkout`), or every line will be blamed on the shallow boundary
commit and look as new as that commit.

## Exit status

| Code | Meaning |
|------|---------|
| 0 | Success, including when no markers were found. |
| 1 | `--fail-older-than` found an older marker. |
| 2 | Error: not inside a git work tree, bad flag or path, git failed. |

If a single file cannot be read or `git blame` fails for it, its markers
are skipped with a warning on stderr, and the exit status is not affected.

## How it works

`todo-age` lists files with `git ls-files -z`, reads each one, and runs
`git blame --porcelain` once per file that has markers, with one `-L` range
per run of marked lines, or for the whole file when there are more than a
thousand such runs. Files are blamed in parallel, one `git` process
per CPU. Ages come from the author date of the commit that last touched the
line, so reformatting a line makes it look new; `git blame` ignores commits
listed in `blame.ignoreRevsFile`, which is the usual fix for that.

## Configuration and platforms

There is no configuration file and no environment variable of its own;
put your usual flags in a shell alias. `git` runs with your environment and
config, so settings such as `blame.ignoreRevsFile` apply.

`todo-age` builds and runs on Linux and macOS, and anywhere else Go and
`git` do.

## License

MIT
