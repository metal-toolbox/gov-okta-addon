FROM gcr.io/distroless/static

COPY ./bin/gov-okta-addon /addon

# Run the web service on container startup.
ENTRYPOINT ["/addon"]
CMD ["serve"]
