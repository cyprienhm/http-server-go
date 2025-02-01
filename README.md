# http-server-go

An HTTP server, written in go.

## Quick start

Run the server on port 4221:

```sh
go run cmd/http-server/main.go
```

To serve files from a directory:

```sh
go run cmd/http-server/main.go --directory /path/to/dir/
```

## Supported endpoints

### GET /echo/text

Echo back `text`.

### GET /files/filepath

If the server has been started with `--directory`, tries to serve the file in
the specified directory at `filepath`.

### POST /files/filepath

If the server has been started with `--directory`, write the request body to
the file in the specified directory at `filepath`.

### GET /user-agent

Respond with the client user-agent.

## Some supported features

- Accepts request which contain gzip as Accepted-Encoding.
