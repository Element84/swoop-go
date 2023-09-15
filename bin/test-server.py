#!/usr/bin/env python
import json
import os
import sys

from http.server import BaseHTTPRequestHandler, HTTPServer


config = {}

class Server(HTTPServer):
    allow_reuse_address = True

class Handler(BaseHTTPRequestHandler):
    def respond(self, status: int, body: str) -> None:
        resp = body.encode()
        self.send_response(status)

        self.send_header('Content-Length', str(len(resp)))
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()

        self.wfile.write(resp)

    def do_POST(self):
        content_len = int(self.headers.get('Content-Length'))
        body = self.rfile.read(content_len)

        try:
            json_body = json.loads(body)
            self.respond(**config[json_body["id"]])
        except:
            self.respond(400, "invalid request")


def main() -> None:
    global config
    config = json.load(sys.stdin)

    server_address = ('', int(os.getenv('TEST_SERVER_PORT', 7986)))

    with Server(server_address, Handler) as server:
        server.serve_forever()


if '__main__' == __name__:
    sys.exit(main())
