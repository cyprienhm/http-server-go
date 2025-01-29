#!/bin/bash

curl -v -H "Accept-Encoding: gzip" http://localhost:4221/echo/abc | hexdump -C
