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
AGE   DATE        AUTHOR       TAG         LOCATION              TEXT
412d  2025-04-01  Maria Lopez  TODO        server/handler.go:88  validate the redirect target
97d   2026-02-10  Sam Okafor   FIXME(sam)  store/cache.go:14     eviction ignores TTL
3d    2026-05-15  Maria Lopez  HACK        cmd/main.go:41        remove once v2 config ships
```

It looks for `TODO`, `FIXME`, `HACK` and `XXX` in tracked files, skipping
binary files and files over 1 MiB.

```sh
todo-age src/ cmd/              # only files under these paths
todo-age --tag FIXME            # only FIXMEs
todo-age --tags TODO,FIXME,BUG  # look for a different set of markers
todo-age --format json          # also csv and ndjson
todo-age --exclude 'vendor/**,*.min.js'
todo-age --older-than 90d       # ages: h, d, w, mo, y, or combined as 1y6mo
todo-age --newer-than 2w
todo-age --author '(?i)maria'   # regex on author name or email
```

Exclude patterns without a slash (`*.min.js`, `vendor`) match a file or
directory name at any depth; patterns with a slash (`vendor/**`,
`web/*.css`) match from the top of the scanned tree, and `**` spans any
number of directories. Use `--max-size` to change the 1 MiB cutoff.

Each row has `file`, `line`, `tag`, `text`, `owner` (from `TODO(owner)`),
`author`, `email`, `commit`, `date`, `age_days` and `uncommitted`.

## License

MIT
