#!/usr/bin/env sh

cd $(dirname $0)/..

grep -nER "\breflect\.DeepEqual\b" --include '*.go' --exclude-dir vendor cmd codegen pkg
if [ $? -eq 0 ]; then
  echo
  echo "Please replace calls to reflect.DeepEqual with equality.Semantic.DeepEqual."
  exit 1
fi
