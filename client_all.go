package requester

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"runtime/debug"
)

// Define methods and encodings type
type Methods string
type Encoding string

// Define a type for multipart files
type File struct {
	FileName  string    // Name of the file
	FieldName string    // Name of the field
	Reader    io.Reader // Reader of the file
}

// Define standard error messages
var (
	ErrNoRequest  = "no request has been set"                  // Error message for when no request has been set
	ErrNoCallback = "no callback has been set"                 // Error message for when no callback has been set
	ErrNoEncoding = "no encoding has been set or is not valid" // Error message for when no encoding has been set
)

// Define request methods
const (
	GET     Methods = "GET"     // GET method
	POST    Methods = "POST"    // POST method
	PUT     Methods = "PUT"     // PUT method
	PATCH   Methods = "PATCH"   // PATCH method
	DELETE  Methods = "DELETE"  // DELETE method
	OPTIONS Methods = "OPTIONS" // OPTIONS method
	HEAD    Methods = "HEAD"    // HEAD method
	TRACE   Methods = "TRACE"   // TRACE method

)

// Define methods of encoding
const (
	FORM_URL_ENCODED Encoding = "application/x-www-form-urlencoded" // FORM_URL_ENCODED encoding
	MULTIPART_FORM   Encoding = "multipart/form-data"               // MULTIPART_FORM encoding
	JSON             Encoding = "json"                              // JSON encoding
	XML              Encoding = "xml"                               // XML encoding
)

// APIClient is a client that can be used to make requests to a server.
func NewAPIClient() *APIClient {
	return &APIClient{
		client:        &http.Client{},
		alwaysRecover: true,
	}
}

// Get a request for the specified url
func (c *APIClient) getRequest(method Methods, url string) *http.Request {
	request, err := http.NewRequest(string(method), url, nil)
	if err != nil {
		panic(err)
	}
	return request
}

// Initialize a GET request
func (c *APIClient) Get(url string) *APIClient {
	c.request = c.getRequest(GET, url)
	return c
}

// Initialize a POST request
func (c *APIClient) Post(url string) *APIClient {
	c.request = c.getRequest(POST, url)
	return c
}

// Initialize a PUT request
func (c *APIClient) Put(url string) *APIClient {
	c.request = c.getRequest(PUT, url)
	return c
}

// Initialize a PATCH request
func (c *APIClient) Patch(url string) *APIClient {
	c.request = c.getRequest(PATCH, url)
	return c
}

// Initialize a DELETE request
func (c *APIClient) Delete(url string) *APIClient {
	c.request = c.getRequest(DELETE, url)
	return c
}

// Initialize a OPTIONS request
func (c *APIClient) Options(url string) *APIClient {
	c.request = c.getRequest(OPTIONS, url)
	return c
}

// Initialize a HEAD request
func (c *APIClient) Head(url string) *APIClient {
	c.request = c.getRequest(HEAD, url)
	return c
}

// Initialize a TRACE request
func (c *APIClient) Trace(url string) *APIClient {
	c.request = c.getRequest(TRACE, url)
	return c
}

// Add form data to the request
func (c *APIClient) WithData(formData map[string]string, encoding Encoding, file ...File) *APIClient {
	if c.request == nil {
		c.errorFunc(errors.New(ErrNoRequest))
	}

	switch encoding {
	case JSON:
		c.request.Header.Set("Content-Type", string(JSON))
		buf := new(bytes.Buffer)
		var err = json.NewEncoder(buf).Encode(formData)
		c.errorFunc(err)
		c.request.Body = io.NopCloser(buf)

	case FORM_URL_ENCODED:
		c.request.Header.Set("Content-Type", string(FORM_URL_ENCODED))
		var formValues = url.Values{}
		for k, v := range formData {
			formValues.Add(k, v)
		}
		c.request.Body = io.NopCloser(bytes.NewBufferString(formValues.Encode()))

	case MULTIPART_FORM:
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for k, v := range formData {
			writer.WriteField(k, v)
		}
		for _, f := range file {
			part, err := writer.CreateFormFile(f.FieldName, f.FileName)
			c.errorFunc(err)
			_, err = io.Copy(part, f.Reader)
			c.errorFunc(err)
		}
		c.request.Header.Set("Content-Type", writer.FormDataContentType())
		c.request.Body = io.NopCloser(body)
	default:
		c.errorFunc(errors.New(ErrNoEncoding))
	}
	return c
}

// Make a request with url query parameters
func (c *APIClient) WithQuery(query map[string]string) *APIClient {
	if c.request == nil {
		c.errorFunc(errors.New(ErrNoRequest))
	}
	q := c.request.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	c.request.URL.RawQuery = q.Encode()
	return c
}

// Add headers to the request
func (c *APIClient) WithHeaders(headers map[string]string) *APIClient {
	if c.request == nil {
		c.errorFunc(errors.New(ErrNoRequest))
	}
	for k, v := range headers {
		c.request.Header.Set(k, v)
	}
	return c
}

// Add a HTTP.Cookie to the request
func (c *APIClient) WithCookie(cookie *http.Cookie) *APIClient {
	if c.request == nil {
		c.errorFunc(errors.New(ErrNoRequest))
	}
	c.request.AddCookie(cookie)
	return c
}

// Change the request before it is made
// This is useful for adding headers, cookies, etc.
func (c *APIClient) ChangeRequest(cb func(rq *http.Request)) *APIClient {
	if c.request == nil {
		c.clientErr(errors.New(ErrNoRequest))
	} else if cb == nil {
		c.clientErr(errors.New(ErrNoCallback))
	}
	cb(c.request)
	return c
}

// Make a request and return the response body decoded into the specified parameter. -> APIClient.Do  -> APIClient.exec
func (c *APIClient) DoDecodeTo(decodeTo interface{}, encoding Encoding, cb func(resp *http.Response, strct interface{})) {
	var newCallback = func(resp *http.Response) {
		switch encoding {
		case JSON:
			if err := json.NewDecoder(resp.Body).Decode(decodeTo); err != nil {
				c.clientErr(err)
			}
		case XML:
			if err := xml.NewDecoder(resp.Body).Decode(decodeTo); err != nil {
				c.clientErr(err)
			}
		default:
			c.clientErr(errors.New(ErrNoEncoding))
		}

		if cb != nil {
			cb(resp, decodeTo)
		}
	}
	c.Do(newCallback)
}

// Set the callback function for when an error occurs
func (c *APIClient) OnError(cb func(err error) bool) *APIClient {
	c.errorFunc = cb
	return c
}

// Do not reccover when an error occurs
func (c *APIClient) NoRecover() *APIClient {
	c.alwaysRecover = false
	return c
}

func (c *APIClient) clientErr(err error) {
	if c.alwaysRecover {
		defer PrintRecover()
	}
	if err != nil {
		if c.errorFunc != nil {
			if c.errorFunc(err) {
				panic(err)
			} else {
				return
			}
		} else {
			panic(err)
		}
	}
}

// Recover from a panic and print the stack trace
func PrintRecover() any {
	if r := recover(); r != nil {
		println(string(debug.Stack()))
		println("///////////////////////////////////////////")
		println("///")
		println(fmt.Sprintf("///	%v", r))
		println("///")
		println("///////////////////////////////////////////")
		return r
	}
	return nil
}
