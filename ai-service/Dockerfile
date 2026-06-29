FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends wget && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Modelo baixado em runtime; mapeado via volume model_cache
ENV EMBED_MODEL=sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2
ENV EMBED_PORT=8082

EXPOSE 8082

CMD ["python3", "embed_server.py"]
