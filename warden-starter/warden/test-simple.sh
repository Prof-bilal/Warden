#!/bin/bash
echo "Test script running"
echo "Args: $@"
echo "Testing file access:"
ls /tmp/ 2>&1 || echo "Cannot access /tmp"
