#!/usr/bin/env sh

cd $(dirname $0)/../..

grep -nER "\breflect\.DeepEqual\b" --include '*.go' --exclude-dir vendor api
if [ $? -eq 0 ]; then
  echo
  echo "Please replace calls to reflect.DeepEqual with equality.Semantic.DeepEqual."
  exit 1
fi
