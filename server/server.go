package main

import (
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

const HTTP_VER = "HTTP/1.1"
const CRLF = "\r\n"

const RESPONSE_200 = "200 OK"
const RESPONSE_201 = "201 Created"
const RESPONSE_400 = "400 Bad Request"
const RESPONSE_404 = "404 Not Found"

type HTTPRequest struct {
	method        string
	requestTarget string
	protocol      string
	headers       map[string]string
	body          string
}
type HTTPResponse struct {
	status  string
	headers map[string]string
	body    string
}

var directory string

func main() {
	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args)-1; i += 2 {
			if os.Args[i] == "--dir" || os.Args[i] == "--directory" {
				directory = os.Args[i+1]
			}
		}
	}

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	running := true

	for running {
		connection, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
		}

		go processRequest(connection)
	}
}

func parseRequest(request []byte) HTTPRequest {
	var httpRequest HTTPRequest
	httpRequest.headers = make(map[string]string)
	strBuffer := string(request)
	header, body, _ := strings.Cut(strBuffer, CRLF+CRLF)
	splitHeaders := strings.Split(header, CRLF)
	firstLine := splitHeaders[0]
	firstLineTokens := strings.Split(firstLine, " ")
	if len(firstLineTokens) < 3 {
		fmt.Println("Could not parse first line.")
		return httpRequest
	}
	httpRequest.method = firstLineTokens[0]
	httpRequest.requestTarget = firstLineTokens[1]
	httpRequest.protocol = firstLineTokens[2]

	for _, headerLine := range splitHeaders[1:] {
		headerKey, headerValue, found := strings.Cut(headerLine, ":")
		headerKey = strings.Trim(headerKey, " ")
		headerValue = strings.Trim(headerValue, " ")
		if !found {
			continue
		}
		httpRequest.headers[headerKey] = headerValue
	}
	httpRequest.body = body

	fmt.Println(httpRequest)
	return httpRequest
}

func processRequest(connection net.Conn) {
	defer connection.Close()
	buffer := make([]byte, 2048)
	_, err := connection.Read(buffer)
	if err != nil {
		fmt.Println("Could not read from net.Conn", err.Error())
	}

	request := parseRequest(buffer)

	var response HTTPResponse
	response.headers = make(map[string]string)
	if userAgent, ok := request.headers["User-Agent"]; ok {
		response.headers["User-Agent"] = userAgent
	}

	switch request.method {
	case "GET":
		processGET(request, &response)
	case "POST":
		processPOST(request, &response)
	default:
		fmt.Println("Did not recognize method: ", request.method)
		response.status = RESPONSE_404
	}

	acceptedEncodings, ok := request.headers["Accept-Encoding"]
	if ok {
		listEncodings := strings.Split(acceptedEncodings, ",")
		for i := range listEncodings {
			listEncodings[i] = strings.TrimSpace(listEncodings[i])
		}
		if slices.Contains(listEncodings, "gzip") {
			response.headers["Content-Encoding"] = "gzip"
			var buffer strings.Builder
			gz := gzip.NewWriter(&buffer)
			_, err := gz.Write([]byte(response.body))
			if err != nil {
				fmt.Println("Could not zip")
				os.Exit(1)
			}
			gz.Close()
			resultEncodedString := buffer.String()
			resultEncodedString = strings.Trim(resultEncodedString, " ")
			response.headers["Content-Length"] = strconv.Itoa(len(resultEncodedString))
			response.body = resultEncodedString
		}
	}
	sendHTTPResponse(response, connection)
}

func processPOST(request HTTPRequest, response *HTTPResponse) {
	fmt.Println("Processing POST")

	requestedURI := request.requestTarget
	fmt.Println("Requested URL: ", requestedURI)

	filenameRegex, _ := regexp.Compile("/files/.*")
	if filenameRegex.Match([]byte(requestedURI)) {
		requestedFile, _ := strings.CutPrefix(requestedURI, "/files/")

		contentLength, err := strconv.Atoi(request.headers["Content-Length"])
		if err != nil || contentLength == 0 {
			response.status = RESPONSE_400
			return
		}
		fmt.Println("going to write exactly", contentLength, "bytes")

		err = os.WriteFile(directory+"/"+requestedFile, []byte(request.body[:contentLength]), 0666)
		if err != nil {
			response.status = RESPONSE_404
			return
		}
		response.status = RESPONSE_201
		request.body = ""
		return
	}

	response.status = RESPONSE_404
	response.body = ""
}

func processGET(request HTTPRequest, response *HTTPResponse) {
	fmt.Println("Processing GET")

	requestedURI := request.requestTarget
	fmt.Println("Requested URL: ", requestedURI)

	if requestedURI == "/" {
		response.status = RESPONSE_200
		response.body = ""
		return
	}

	echoRegex, _ := regexp.Compile("/echo/.*")
	if echoRegex.Match([]byte(requestedURI)) {
		echoBack, _ := strings.CutPrefix(requestedURI, "/echo/")
		response.status = RESPONSE_200
		response.headers["Content-Type"] = "text/plain"
		response.headers["Content-Length"] = strconv.Itoa(len(echoBack))
		response.body = echoBack
		return
	}

	userAgentRegex, _ := regexp.Compile("/user-agent?")
	if userAgentRegex.Match([]byte(requestedURI)) {
		userAgent := request.headers["User-Agent"]
		response.status = RESPONSE_200
		response.headers["Content-Type"] = "text/plain"
		response.headers["Content-Length"] = strconv.Itoa(len(userAgent))
		response.body = userAgent
		return
	}

	filenameRegex, _ := regexp.Compile("/files/.*")
	if filenameRegex.Match([]byte(requestedURI)) {
		requestedFile, _ := strings.CutPrefix(requestedURI, "/files/")

		fileContents, err := os.ReadFile(directory + "/" + requestedFile)
		if err != nil {
			response.status = RESPONSE_404
			return
		}
		response.status = RESPONSE_200
		response.headers["Content-Type"] = "application/octet-stream"
		response.headers["Content-Length"] = strconv.Itoa(len(fileContents))
		response.body = string(fileContents)
		return
	}

	response.status = RESPONSE_404
	response.body = ""
}

func writeResponse(statusCode string, conn net.Conn) {
	var response strings.Builder

	response.WriteString(HTTP_VER)
	response.WriteString(" ")
	response.WriteString(statusCode)
	response.WriteString(CRLF)
	response.WriteString(CRLF)

	conn.Write([]byte(response.String()))
}

func sendHTTPResponse(response HTTPResponse, conn net.Conn) {
	var toSend strings.Builder

	toSend.WriteString(HTTP_VER)
	toSend.WriteString(" ")
	toSend.WriteString(response.status)
	toSend.WriteString(CRLF)
	for k, v := range response.headers {
		toSend.WriteString(k)
		toSend.WriteString(": ")
		toSend.WriteString(v)
		toSend.WriteString(CRLF)
	}
	toSend.WriteString(CRLF)
	toSend.WriteString(response.body)
	fmt.Println("Sending response:-")
	fmt.Println(toSend.String())
	fmt.Println("-")

	conn.Write([]byte(toSend.String()))

}
