#!/usr/bin/env python3
"""Minimal fake usbipd for the README `list -r` smoke test."""

from __future__ import annotations

import argparse
import socket


OP_REQ_DEVLIST = bytes.fromhex("0111800500000000")
OP_REP_DEVLIST_EMPTY = bytes.fromhex("011100050000000000000000")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Serve an empty USB/IP devlist reply for local smoke tests.",
    )
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", default=19000, type=int)

    return parser.parse_args()


def main() -> None:
    args = parse_args()
    address = (args.host, args.port)

    with socket.socket() as sock:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        sock.bind(address)
        sock.listen()
        print(f"fake usbipd listening on {args.host}:{args.port}", flush=True)

        while True:
            conn, peer = sock.accept()
            with conn:
                request = conn.recv(len(OP_REQ_DEVLIST))
                print(f"accepted {peer}, request={request.hex()}", flush=True)
                if request == OP_REQ_DEVLIST:
                    conn.sendall(OP_REP_DEVLIST_EMPTY)


if __name__ == "__main__":
    main()
