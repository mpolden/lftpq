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
    "LftpPath": "lftp",
    "LftpGetCmd": "mirror"
  },
  "Sites": [
    {
      "Name": "foo",
      "Dir": "/dir",
      "LocalDir": "/tmp/{{ .Name }}/S{{ .Season }}/",
      "SkipSymlinks": true,
      "ParseTVShow": true,
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
