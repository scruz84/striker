#!/bin/sh

#Took from https://www.bencode.net/posts/react-build/
cp -f /usr/share/nginx/html/config.js /tmp
if [ -n "$API_URI" ]; then
sed -i -e "s|REPLACE_API_ENDPOINT|$API_URI|g" /tmp/config.js
fi
cat /tmp/config.js > /usr/share/nginx/html/config.js