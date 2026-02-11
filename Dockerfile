FROM gcr.io/distroless/static:nonroot

# `nonroot` coming from distroless
USER 65532:65532

# pass in name as --build-arg
ARG NAME

COPY --chmod=755 ./bin/${NAME} /addon

# Run the web service on container startup.
ENTRYPOINT ["/addon"]
CMD ["serve"]
