#!/bin/sh
fc-cache -fv /usr/local/share/fonts/project
fc-cache -fv /usr/local/share/fonts/api
exec /usr/local/bin/server
