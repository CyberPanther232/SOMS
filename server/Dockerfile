FROM python:3.12-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Ensure data directory exists for sqlite
RUN mkdir -p data

ENV SOMS_SERVER_HOST=0.0.0.0
ENV SOMS_SERVER_PORT=5000
ENV SOMS_DEBUG=False

EXPOSE 5000

CMD ["python", "main.py"]
