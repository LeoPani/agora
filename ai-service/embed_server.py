#!/usr/bin/env python3
"""
Servidor de embedding leve (HTTP, porta 8082).
Carrega o modelo uma vez e serve vetores sob demanda para a API Go.

GET /embed?text=agricultura+sustentavel
  → {"embedding": [0.12, -0.34, ...]}   (384 floats)

GET /health
  → {"status": "ok", "model": "...", "dims": 384}

Uso:
  python3 embed_server.py
  python3 embed_server.py --model LeoPani/patentbert-br --port 8082
"""

import argparse
import json
import os
import sys
import urllib.parse
from http.server import BaseHTTPRequestHandler, HTTPServer

MODEL_NAME = os.getenv(
    "EMBED_MODEL",
    "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"
)
PORT = int(os.getenv("EMBED_PORT", "8082"))

model = None


def load_model(name: str):
    global model
    try:
        from sentence_transformers import SentenceTransformer
    except ImportError:
        print("Instalando sentence-transformers...")
        os.system(f"{sys.executable} -m pip install sentence-transformers -q")
        from sentence_transformers import SentenceTransformer
    print(f"[embed_server] carregando {name}...")
    model = SentenceTransformer(name)
    dims = model.get_sentence_embedding_dimension()
    print(f"[embed_server] pronto — {dims} dims")
    return dims


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass  # silencia logs HTTP normais

    def send_json(self, code: int, data: dict):
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        params = urllib.parse.parse_qs(parsed.query)

        if parsed.path == "/health":
            dims = model.get_sentence_embedding_dimension() if model else 0
            self.send_json(200, {"status": "ok", "model": MODEL_NAME, "dims": dims})
            return

        if parsed.path == "/embed":
            text = params.get("text", [""])[0]
            if not text:
                self.send_json(400, {"error": "text is required"})
                return
            if model is None:
                self.send_json(503, {"error": "model not loaded"})
                return
            vec = model.encode([text], normalize_embeddings=True)[0]
            self.send_json(200, {"embedding": vec.tolist()})
            return

        self.send_json(404, {"error": "not found"})


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", default=MODEL_NAME)
    parser.add_argument("--port",  type=int, default=PORT)
    args = parser.parse_args()

    load_model(args.model)

    server = HTTPServer(("0.0.0.0", args.port), Handler)
    print(f"[embed_server] escutando em http://0.0.0.0:{args.port}")
    server.serve_forever()


if __name__ == "__main__":
    main()
