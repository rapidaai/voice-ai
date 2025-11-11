#!/bin/bash

# Start Celery worker in the background and capture its PID
celery -A app.celery_worker.celery_app worker --loglevel=info --concurrency=8 &
CELERY_WORKER_PID=$!

# Start Flower for monitoring Celery tasks in the background and capture its PID
celery -A app.celery_worker.celery_app flower --port=5555 &
FLOWER_PID=$!

# Define a cleanup function to kill both processes
cleanup() {
  echo "Stopping Celery worker and Flower..."
  kill $CELERY_WORKER_PID
  kill $FLOWER_PID
}

# Trap the EXIT signal to call the cleanup function when the script exits
trap cleanup EXIT

# Wait for all background jobs (this keeps the script running)
wait