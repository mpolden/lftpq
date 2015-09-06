# lftpq

[![Build Status](https://travis-ci.org/martinp/lftpq.png)](https://travis-ci.org/martinp/lftpq)

A queue generator for [lftp](http://lftp.yar.ru).

## Usage

```
# lftpq -h
Usage:
  lftpq [OPTIONS]

Application Options:
  -f, --config=FILE    Path to config (~/.lftpqrc)
  -n, --dryrun         Print generated queue and exit without executing lftp
  -t, --test           Test and print config
  -q, --quiet          Only print errors
  -v, --verbose        Verbose output

Help Options:
  -h, --help           Show this help message
```

## Example config

```json
{
  "Client": {
    "Path": "lftp",
    "GetCmd": "mirror"
  },
  "Sites": [
    {
      "Name": "foo",
      "Dir": "/dir",
      "LocalDir": "/tmp/{{ .Name }}/S{{ .Season }}/",
      "SkipSymlinks": true,
      "SkipExisting": true,
      "Priorities": [
        "^important",
        "^less\\.important"
      ],
      "Parser": "show",
      "MaxAge": "24h",
      "Patterns": [
        "^Dir1",
        "^Dir2"
      ],
      "Filters": [
        "(?i)incomplete"
      ]
    }
  ]
}
```

## Configuration options

`Client` holds configuration related to lftp.

`Path` sets the path to the lftp executable (if only the base name is given,
`PATH` will be used for lookup).

`GetCmd` sets the lftp command to use when downloading, this can also be an
alias. For example: If you have `alias m "mirror --only-missing"` in your
`.lftprc`, then `LftpGetCmd` can be set to `m`.

`Sites` holds the configuration for each individual site.

`Name` is the bookmark or URL of the site. This is passed to the `open` command in lftp.

`Dir` is the remote directory used to generate the queue.

`LocalDir` is the local directory where files should be downloaded. This can be
a template. When the `show` parser is used, the following template variables are
available:

Variable  | Description                                    | Example
--------- | ---------------------------------------------- | -------
`Name`    | Name of the show                               | `The.Wire`
`Season`  | Show season                                    | `01`
`Episode` | Show episode                                   | `05`
`Release` | Release/directory name                         | `The.Wire.S01E05.720p.BluRay.X264`

When using the `movie` parser, the following variables are available:

Variable  | Description                                    | Example
--------- | ---------------------------------------------- | -------
`Name`    | Movie name                                     | `Apocalypse.Now`
`Year`    | Production year                                | `1979`
`Release` | Release/directory name                         | `Apocalypse.Now.1979.720p.BluRay.X264`

`SkipSymlinks` determines whether to ignore symlinks when generating the queue.

`SkipExisting` determines whether to ignore non-empty directories that already
exist in `LocalDir`.

`Priorities` is a list of patterns used to deduplicate directories which contain
the same media (e.g. same show, season and episode, but different release).
Directories are deduplicated based on the order of matching patterns, where the
earliest match is given the highest weight. For example, if the items
`Foo.1.important` and `Foo.1.less.important` are determined to be the same
media, then given the priorities in the example above, `Foo.1.important` would
be kept and `Foo.2.less.Important` would be removed from the queue.

`Parser` sets the parser to use when parsing media. Valid values are `show`,
`movie` or empty string (disable parsing).

`MaxAge` sets the maximum age of directories to consider for the queue. If a
directory is older than `MaxAge`, it will always be excluded. `MaxAge` has
precedence over `Patterns` and `Filters`.

`Patterns` is a list of patterns (regular expressions) used when including
directories. A directory matching any of these patterns will be included in the
queue.

`Filters` is a list of patterns used when excluding directories. A directory
matching any of these patterns will be excluded from the queue. `Filters` has
precedence over `Patterns`.
