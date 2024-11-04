FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-freshservice"]
COPY baton-freshservice /