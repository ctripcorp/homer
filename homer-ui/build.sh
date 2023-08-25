#!/bin/bash
mkdir build
npm install && npm install -g @angular/cli && npm run build
cp -R ./dist/homer-ui/* ./dist
rm -rf ./dist/homer-ui
cat src/VERSION.ts | egrep -o '[0-9].[0-9].[0-9]+' > dist/VERSION
tar zcvf homer-ui.tgz dist/
mv -f homer-ui.tgz ./build
echo 'done!'
