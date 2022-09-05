FROM btwiuse/arch:golang

WORKDIR /app

COPY . ./

RUN go mod tidy

RUN go mod download

CMD go run ./cmd/demo
