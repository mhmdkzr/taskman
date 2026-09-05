# Grep Tool

`grep` searches a file or directory with a regular expression and returns matching lines. It accepts an optional file glob and result limit; searches exclude `.git` and hidden files are included.

Example input:

```json
{"pattern":"TODO","path":"internal","include":"*.go","max_results":20}
```
