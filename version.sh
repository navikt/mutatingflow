#!/bin/sh
echo $(date "+%Y-%m-%d")-$(git --no-pager log -1 --pretty=%h)
