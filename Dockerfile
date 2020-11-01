FROM vector-search-go:latest

RUN go mod tidy && go build -o /usr/bin/app ./main.go 
RUN go build -o /usr/bin/run_prep_data ./data/run_prep_data.go

EXPOSE 8080
ENTRYPOINT [ "sh" ]

# CMD [ "app" ]


