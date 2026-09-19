#!/bin/sh
# Rewrites the placeholders baked into config.js at container start, from real environment
# variables, so the same built image can be pointed at different backends/OAuth clients.

cp -f /usr/share/nginx/html/config.js /tmp/config.js
if [ -n "$API_URI" ]; then
  sed -i -e "s|REPLACE_API_ENDPOINT|$API_URI|g" /tmp/config.js
fi
if [ -n "$GOOGLE_OAUTH_CLIENT_ID" ]; then
  sed -i -e "s|REPLACE_GOOGLE_OAUTH_CLIENT_ID|$GOOGLE_OAUTH_CLIENT_ID|g" /tmp/config.js
fi
if [ -n "$VAPID_PUBLIC_KEY" ]; then
  sed -i -e "s|REPLACE_VAPID_PUBLIC_KEY|$VAPID_PUBLIC_KEY|g" /tmp/config.js
fi
cat /tmp/config.js > /usr/share/nginx/html/config.js
