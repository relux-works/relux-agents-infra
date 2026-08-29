#!/usr/bin/env python3
"""A healthy HTTP surface that llama-server's caller cannot tell from the real
thing at the protocol level, but whose answer is chosen by FAKE_MODE.

Used only by test_load_answer_gate.sh to attack load_and_answer.sh. It accepts
llama-server's argv shape so it can be dropped in via LLAMA_SERVER_BIN.

FAKE_MODE:
  right      correct answer, finish_reason stop      -> the gate must ACCEPT
  wrong      confident wrong answer, finish_reason stop
  truncated  correct answer but finish_reason length
  empty      finish_reason stop, empty content
  malformed  HTTP 200 with a body that is not the expected JSON shape
  notjson    HTTP 200 with a body that is not JSON at all
  zerotokens correct answer but usage reports 0 completion tokens
"""
import json
import os
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

MODE = os.environ.get("FAKE_MODE", "right")
port = int(sys.argv[sys.argv.index("--port") + 1])

BODIES = {
    "right": {"choices": [{"finish_reason": "stop",
                           "message": {"content": "The capital of Armenia is Yerevan."}}],
              "usage": {"completion_tokens": 10, "prompt_tokens": 25, "total_tokens": 35}},
    "wrong": {"choices": [{"finish_reason": "stop",
                           "message": {"content": "The capital of Armenia is Baku."}}],
              "usage": {"completion_tokens": 9, "prompt_tokens": 25, "total_tokens": 34}},
    "truncated": {"choices": [{"finish_reason": "length",
                               "message": {"content": "The capital of Armenia is Yerevan"}}],
                  "usage": {"completion_tokens": 96, "prompt_tokens": 25, "total_tokens": 121}},
    "empty": {"choices": [{"finish_reason": "stop", "message": {"content": "   "}}],
              "usage": {"completion_tokens": 0, "prompt_tokens": 25, "total_tokens": 25}},
    "malformed": {"error": {"message": "context shift disabled", "code": 500}},
    "zerotokens": {"choices": [{"finish_reason": "stop",
                                "message": {"content": "The capital of Armenia is Yerevan."}}],
                   "usage": {"completion_tokens": 0, "prompt_tokens": 25, "total_tokens": 25}},
}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, _format, *_args):
        return

    def do_GET(self):
        self.send_response(200 if self.path == "/health" else 404)
        self.end_headers()

    def do_POST(self):
        if MODE == "notjson":
            payload = b"<html>502 Bad Gateway</html>"
            ctype = "text/html"
        else:
            payload = json.dumps(BODIES[MODE]).encode()
            ctype = "application/json"
        self.send_response(200)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


HTTPServer(("127.0.0.1", port), Handler).serve_forever()
