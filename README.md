[![Go Report Card](https://goreportcard.com/badge/github.com/komkom/jsonc)](https://goreportcard.com/report/github.com/komkom/jsonc)

# jsonc

Jsonc is a simplified json format which allows comments and unquoted values delimited by whitespace. A jsonc formatted file can be unambiguously transformed to a json file. Comments will be stripped out and quotes added.

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

## Syntax
Here is a first attempt to formalize the jsonc syntax in [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form).

```
document = type;

type = object | array | basicType;
basicType = nullSymbol | booleanSymbol | string | number;

object = "{" [{property [","] } property] "}";
property = string ":" type;

array = "[" [{type [","]} type] "]";

nullSymbol = "null";
booleanSymbol = "true" | "false";

string = stringToken | stringMultiline | jsonString;

stringToken = "a string containing only digits or letters.";
stringMultiline = "`" {stringSymbol lineBreak {lineBreak}} stringSymbol {lineBreak} "`";
jsonString = "a string as defined by json dot org including surrounding \" ";
lineBreak = "the line break character \n";

number = "a number as defined by json dot org";
```

### Examples

```
{
 // A jsonc example. 
 glossEntry:{
  id:SGML
  sortAs:SGML
  glossTerm:`Standard Generalized Markup language`
  acronym:SGML
  abbrev:`ISO 8879:1986`
  glossDef:{
   para:`A meta-markup language, used to create markup languages such as DocBook.`
   glossSeeAlso:[GML XML]
}}}
```

```
{
 // Another example
 popup:{
  menuitem:[
   {value:New,onclick:`CreateNewDoc()`}
   {value:Open,onclick:`OpenDoc()`}
   {value:Close,onclick:`CloseDoc()`}
  ]
}}
```

The formatting behaviour tries to mimic gofmt. Any suggestions, help, fixes welcome.
