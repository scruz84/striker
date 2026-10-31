#!/bin/sh
# Rewrites the placeholders baked into config.js at container start, from real environment
# variables, so the same built image can be pointed at different backends/OAuth clients and
# operated by different legal entities (the LEGAL_* properties).

cp -f /usr/share/nginx/html/config.js /tmp/config.js

# replace_placeholder NAME VALUE - swaps REPLACE_NAME for VALUE in config.js. The value is first
# escaped for the single-quoted JavaScript string it lands in (backslash, quote) and then for
# sed's replacement side (backslash, ampersand, the | delimiter), because company names and
# postal addresses routinely contain & and | and /. A value with a newline is cut at the first one.
replace_placeholder() {
  name="$1"
  value="$(printf '%s' "$2" | head -n 1 | sed -e 's/\\/\\\\/g' -e "s/'/\\\\'/g" | sed -e 's/[\\&|]/\\&/g')"
  sed -i -e "s|REPLACE_$name|$value|g" /tmp/config.js
}

if [ -n "$API_URI" ]; then replace_placeholder API_ENDPOINT "$API_URI"; fi
if [ -n "$GOOGLE_OAUTH_CLIENT_ID" ]; then replace_placeholder GOOGLE_OAUTH_CLIENT_ID "$GOOGLE_OAUTH_CLIENT_ID"; fi
if [ -n "$VAPID_PUBLIC_KEY" ]; then replace_placeholder VAPID_PUBLIC_KEY "$VAPID_PUBLIC_KEY"; fi
for name in LEGAL_CONTROLLER_NAME LEGAL_CONTACT_EMAIL LEGAL_CONTROLLER_TAX_ID LEGAL_POSTAL_ADDRESS \
  LEGAL_HOSTING_PROVIDER LEGAL_HOSTING_REGION LEGAL_NOTICE_ENABLED; do
  eval "value=\${$name}"
  if [ -n "$value" ]; then replace_placeholder "$name" "$value"; fi
done
cat /tmp/config.js > /usr/share/nginx/html/config.js
