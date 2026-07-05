# todo-age

Lists the TODOs in a git repository, oldest first, with the age and author
of each one according to `git blame`.

Every codebase collects `TODO` comments. Most are fine; some have been
sitting there for three years. `todo-age` finds the old ones.

## Install

```sh
go install github.com/dan2756/todo-age@latest
```

Requires `git` on your `PATH`.

## Usage

Run it anywhere inside a git work tree:

```sh
$ todo-age
AGE     DATE        AUTHOR       TAG         LOCATION              TEXT
1y 1mo  2025-04-01  Maria Lopez  TODO        server/handler.go:88  validate the redirect target
3mo     2026-02-10  Sam Okafor   FIXME(sam)  store/cache.go:14     eviction ignores TTL
3d      2026-05-19  Maria Lopez  HACK        cmd/main.go:41        remove once v2 config ships
```

It looks for `TODO`, `FIXME`, `HACK` and `XXX` in tracked files, skipping
binary files, files over 1 MiB, symlinks and submodules. Lines you have
changed but not committed show up as `Not Committed Yet` with an age of 0,
as do all markers in a repository without commits.

```sh
todo-age src/ cmd/              # only files under these paths
todo-age --tag FIXME            # only FIXMEs
todo-age --tags TODO,FIXME,BUG  # look for a different set of markers
todo-age --format json          # also csv and ndjson
todo-age --exclude 'vendor/**,*.min.js'
todo-age --older-than 90d       # ages: h, d, w, mo, y, or combined as 1y6mo
todo-age --newer-than 2w
todo-age --author '(?i)maria'   # regex on author name or email
todo-age --sort author          # or file; the default is oldest first
todo-age --limit 10             # the ten oldest
```

Exclude patterns without a slash (`*.min.js`, `vendor`) match a file or
directory name at any depth; patterns with a slash (`vendor/**`,
`web/*.css`) match from the top of the scanned tree, and `**` spans any
number of directories. Use `--max-size` to change the 1 MiB cutoff.

Each row has `file`, `line`, `tag`, `text`, `owner` (from `TODO(owner)`),
`author`, `email`, `commit`, `date`, `age_days` and `uncommitted`.

## Summary

`--summary` prints counts instead of the markers themselves. Filters
apply; `--limit` does not.

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
Run CI checkouts with full history (for example `fetch-depth: 0` on GitHub
Actions), or every line will be blamed on the shallow boundary commit.

## Exit status

| Code | Meaning |
|------|---------|
| 0    | Success. |
| 1    | `--fail-older-than` found an older marker. |
| 2    | Error: not a git repository, bad flags, git failed. |

## License

MIT
