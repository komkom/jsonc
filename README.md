# jsonc
[give it a try](https://komkom.github.io/)

## Install

```bash
go get github.com/komkom/jsonc/...
```

## Use

Prints the formatted jsonc file.
```bash
jsonc < somefile.jsonc 
```

Prints the minified json.
```bash
jsonc -m < somefile.jsonc 
```

To format a json.
```bash
jsonc -m < somefile.json
```

The formatting behaviour tries to mimic gofmt. Any suggestions, help, fixes welcome.
