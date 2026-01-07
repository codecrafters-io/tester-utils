#!/bin/sh
# A script that allocates memory in a loop until killed.
# Used to test memory limiting functionality.

data=""
while true; do
    # Allocate ~10MB chunks by appending to a string
    data="$data$(head -c $((10 * 1024 * 1024)) /dev/zero 2>/dev/null | tr '\0' 'x')"
done