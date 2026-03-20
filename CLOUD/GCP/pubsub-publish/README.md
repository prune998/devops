docker  build -t prune/pubsubpublish:v0.0.1 --build-arg VERSION=0.0.1 .

docker run -ti -e "GCP_PROJECT_ID=123" -e PUBSUB_TOPIC_ID=123 prune/pubsubpublish:v0.0.1
