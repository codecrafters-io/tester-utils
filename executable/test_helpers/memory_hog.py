#!/usr/bin/env python3
"""
A script that allocates memory in a loop until killed.
Used to test memory limiting functionality.
"""
import sys

def main():
    data = []
    try:
        while True:
            # Allocate 10MB chunks
            data.append('x' * (10 * 1024 * 1024))
    except MemoryError:
        print("MemoryError caught", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()

