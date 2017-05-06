#!/bin/sh

gopherjs build main.go
rm main.js.map

if [ $# -eq 0 ] 
then
	uglifyjs --compress --mangle -- main.js > .cmp.js
	mv .cmp.js main.js 
fi

mv main.js static/main.js

cp index.html ../../komkom.github.io
cp -r static ../../komkom.github.io
