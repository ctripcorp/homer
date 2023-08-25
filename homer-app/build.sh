#!/bin/bash
mkdir -p ./build/.paas
make modules
make all
cp -f homer-app ./build/
cp -rf ./etc/* ./build/
cp -f ./release/homer-ui.tgz ./build/
cp -rf ./paas_script/* ./build/.paas/