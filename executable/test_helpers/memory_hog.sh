#!/bin/bash
# A script that allocates memory in a loop until killed.
# Used to test memory limiting functionality.

data=""
while true; do
    # Allocate ~10MB chunks by appending to a string
    data+=$(head -c $((10 * 1024 * 1024)) /dev/zero | tr '\0' 'x')
done