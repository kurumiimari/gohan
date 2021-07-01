#!/usr/bin/env bash

set -euo pipefail

if [ -z ${build_version+x} ]; then
  echo "Must specify a build version."
  exit 1
fi

if [ ! -f "changelog-pending.md" ]; then
  echo "Must specify a changelog."
  exit 1
fi

ssh gohan-builder <<EOF
  export PATH=$PATH:/usr/local/go/bin
  cd /var/deploy/build/gohan

  echo "Updating Git repository."
  git fetch origin
  git reset --hard origin/master

  echo "Cross-compiling."
  build_version="$build_version" make build-cross
EOF

ssh gohan-builder "find /var/deploy/build/gohan/bin -name 'gohan*' | | cut -d '/' -f 7" | while read line ; do
  scp gohan-builder:/var/deploy/build/gohan/bin/$line /tmp/$line
  gpg --default-key B8724012 --detach-sign --armor /tmp/$line
  scp /tmp/$line.asc gohan-builder:/var/deploy/build/gohan/bin/$line.asc
  rm /tmp/$line.asc
done

scp changelog-pending.md gohan-builder:/tmp/changelog-pending.md

ssh gohan-builder <<EOF
  cd /var/deploy/build/gohan
  gh release create -F /tmp/changelog-pending.md -R kurumiimari/gohan "$build_version"
  gh release upload --clobber -R kurumiimari/gohan "$build_version" ./bin/*
EOF

