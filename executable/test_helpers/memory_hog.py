#!/usr/bin/env python3
"""
A script that allocates memory in a loop until killed.
Used to test memory limiting functionality.
"""


def main():
    data = []
    while True:
        # Allocate 10MB chunks
        data.append("x" * (10 * 1024 * 1024))


if __name__ == "__main__":
    main()
