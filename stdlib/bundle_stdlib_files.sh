#!/usr/bin/env bash

THISDIR=$(dirname $0)

LIB_CONTENT=()

cat <<EOM | gofmt > $THISDIR/stdlib_gen.go
// Code generated during build process from tl files. DO NOT EDIT.
package stdlib

EOM

for file in "$THISDIR"/*.tl; do
    content=`cat "$THISDIR/$file"`
    filename=`basename "${file%.*}"`
    assoc_array_entry="$filename:$content"
    LIB_CONTENT+=($assoc_array_entry)
    echo "const stdlib_$filename string =\`$content\n\`\n" >> $THISDIR/stdlib_gen.go
done
