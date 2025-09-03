#!/bin/bash
set -e

BOOTSTRAP="${BOOTSTRAP:-broker-1:19092}"
MARKER="/tmp/topics-created"

# Example topic definitions: "topic1:partitions:replication topic2:partitions:replication"
TOPICS="${TOPICS:-participant-resource-update:3:3 asset-resource-delete:3:3 asset-resource-delete-dlq:3:2}"

echo "Creating topics on $BOOTSTRAP..."

while true; do
  success=true

  for entry in $TOPICS; do
    IFS=":" read -r topic partitions replication <<< "$entry"

    echo "Trying to create topic '$topic' with partitions=$partitions replication=$replication..."

    /opt/kafka/bin/kafka-topics.sh \
      --create \
      --topic "$topic" \
      --bootstrap-server "$BOOTSTRAP" \
      --partitions "$partitions" \
      --replication-factor "$replication" \
      --if-not-exists || success=false
  done

  if $success; then
    echo "✅ All topics created (or already exist)."
    touch "$MARKER"
    break
  fi

  echo "Some topic creation failed. Retrying in 5s..."
  sleep 5
done

# Keep the container alive so healthcheck can poll
tail -f /dev/null
