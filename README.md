# jsonc
json config.

## Install

```bash
go get github.com/komkom/jsonc/...
go get -u github.com/jteeuwen/go-bindata/...
go get github.com/tutti-ch/continuous/...
```

## Usage

```bash
jsonc < somefile.jsonc 
```
Prints the formatted jsonc file.

```bash
jsonc -m < somefile.jsonc 
```
Prints the minified json.

```bash
jsonc -m < somefile.json
```
To format a json.

The formatting behaviour tries to mimic gofmt. Any suggestions, help, fixes welcome.
