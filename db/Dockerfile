FROM vector-search-go:latest

RUN go mod tidy && \
    go build -o /usr/bin/run_db ./run_db.go 

EXPOSE 8080
ENTRYPOINT [ "sh" ]

# CMD [ "run_db" ]