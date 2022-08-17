FROM gcr.io/distroless/static

COPY ./gov-okta-addon /addon

# Run the web service on container startup.
ENTRYPOINT ["/addon"]
CMD ["serve"]
