#FROM gcr.io/distroless/static:latest
FROM docker.io/alpine:latest
COPY storage-csi /storage-csi
ENTRYPOINT ["/storage-csi"]
