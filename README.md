[![Go Report Card](https://goreportcard.com/badge/github.com/komkom/jsonc)](https://goreportcard.com/report/github.com/komkom/jsonc)

# jsonc

Jsonc is a simplified json format which allows comments and unquoted values delimited by whitespace. A jsonc formatted file can be transformed to a json file. Comments will be stripped out and quotes added.

Any valid json is also a valid jsonc - but this goes only in one direction.

[give it a try](https://komkom.github.io/)

## Install

```bash
go get github.com/komkom/jsonc/...
```

## Use

### In Code
``` golang
dec, _ := jsonc.NewDecoder(strings.NewReader(`{
  a : "jsonc file" // with comments
}`))

x := struct {
  Key string `json:"a"`
}{}

_ = dec.Decode(&x)
fmt.Printf("%v\n", x)
```

### As CLI

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
