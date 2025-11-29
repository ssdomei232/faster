#!/bin/bash

# 把这里的ID替换为上一步中拿到的ID
id=9

# generate 1panel token
clientToken=$PANEL_API_KEY
timestamp=$(date +%s)
input="1panel$clientToken$timestamp"
token=$(echo -n "$input" | md5sum | awk '{print $1}')

# delete file
curl --location --request POST "$PANEL_URL/api/v2/files/del?operateNode=undefined" \
-H "1Panel-Token: $token" \
-H "1Panel-Timestamp: $timestamp" \
--header 'Content-Type: application/json' \
--data-raw '{"path":"/opt/apps/'"$APP_NAME"'/main","isDir":false,"forceDelete":true}'

# updaload file
curl --location --request POST "$PANEL_URL/api/v2/files/upload" \
-H "1Panel-Token: $token" \
-H "1Panel-Timestamp: $timestamp" \
--form 'path=/opt/apps/'"$APP_NAME" \
--form 'file=@main'

# restart runtime 
curl --location --request POST "$PANEL_URL/api/v2/runtimes/operate" \
-H "1Panel-Token: $token" \
-H "1Panel-Timestamp: $timestamp" \
--header 'Content-Type: application/json' \
--data-raw '{
  "ID": '"$id"',
  "operate": "restart"
}'