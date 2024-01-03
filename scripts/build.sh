PUSH=${PUSH:-"0"}
TAG=${TAG:-"latest"}
IMAGE=git.d464.sh/infra/storage-csi:$TAG

CGO_ENABLED=0 go build || exit 1
docker build -t $IMAGE -f Containerfile .

if [ "$PUSH" -eq "1" ]; then
	docker push $IMAGE
fi
