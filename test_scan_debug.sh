#!/bin/bash
# Test the actual SQL query to see what's happening
cd ~/.openclaw/workspace/zrp
go test -v -run "TestHandleAdvancedSearch_DevicesAndNCRsAndPOs/Search_devices" 2>&1 | grep -A 20 "Search_devices"
