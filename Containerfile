FROM gcr.io/distroless/static:latest
COPY storage-csi /storage-csi
ENTRYPOINT ["/storage-csi"]
