FROM gcr.io/distroless/static:nonroot

# `nonroot` coming from distroless
USER 65532:65532

COPY ./bin/gov-okta-addon /addon

# Run the web service on container startup.
ENTRYPOINT ["/addon"]
CMD ["serve"]
