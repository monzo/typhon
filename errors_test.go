package typhon

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/monzo/terrors"
	"github.com/monzo/terrors/stack"
	"github.com/stretchr/testify/assert"
)

func TestErrorFilter(t *testing.T) {
	var nilReq Request
	tests := []struct {
		name    string
		req     Request
		svc     Service
		wantRsp Response
	}{
		{
			"nil request",
			nilReq,
			Service(func(req Request) Response {
				return req.Response(map[string]string{"b": "a"})
			}),
			Response{
				&http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       &bufCloser{*bytes.NewBufferString(`"b":"a"`)},
				},
				nil,
				&nilReq,
				false,
			},
		},
		{
			"request with foo error",
			Request{
				err: errors.New("foo error"),
			},
			Service(func(req Request) Response {
				return req.Response(map[string]string{"b": "a"})
			}),
			Response{
				&http.Response{
					StatusCode: http.StatusInternalServerError,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
						"Terror":       []string{"1"},
					},
					Body: &bufCloser{*bytes.NewBufferString(`"code":"internal_service","message":"foo error"`)},
				},
				errors.New("foo error"),
				&Request{
					err: errors.New("foo error"),
				},
				false,
			},
		},
		{
			"request with empty error",
			Request{
				err: errors.New(""),
			},
			Service(func(req Request) Response {
				return Response{}
			}),
			Response{
				&http.Response{
					StatusCode: http.StatusInternalServerError,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
						"Terror":       []string{"1"},
					},
					Body: &bufCloser{*bytes.NewBufferString(`"code":"internal_service"`)},
				},
				errors.New("Response error (500)"),
				&Request{
					err: errors.New(""),
				},
				false,
			},
		},
		{
			"service return empty response",
			Request{},
			Service(func(req Request) Response {
				return Response{}
			}),
			Response{
				&http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{},
				},
				nil,
				&Request{
					err: nil,
				},
				false,
			},
		},
		{
			"service return response with 404 no Terror header",
			Request{},
			Service(func(req Request) Response {
				return Response{
					Response: &http.Response{
						Header:     http.Header{},
						StatusCode: http.StatusNotFound,
						Body:       &bufCloser{*bytes.NewBufferString(`"message":"foo not found"`)},
					},
					Error:   nil,
					Request: &Request{},
				}
			}),
			Response{
				&http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{},
				},
				errors.New(`"message":"foo not found"`),
				&Request{
					err: nil,
				},
				false,
			},
		},
		{
			"service return response with 404 with Terror header and no Terror body",
			Request{},
			Service(func(req Request) Response {
				return Response{
					Response: &http.Response{
						Header:     http.Header{"Terror": []string{"1"}},
						StatusCode: http.StatusNotFound,
						Body:       &bufCloser{*bytes.NewBufferString("I am bad boy")},
					},
					Error:   nil,
					Request: &Request{},
				}
			}),
			Response{
				&http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{"Terror": []string{"1"}},
				},
				errors.New("I am bad boy"),
				&Request{
					err: nil,
				},
				false,
			},
		},
		{
			"service return response with 404 with Terror header and Terror body",
			Request{},
			Service(func(req Request) Response {
				return Response{
					Response: &http.Response{
						Header:     http.Header{"Terror": []string{"1"}},
						StatusCode: http.StatusNotFound,
						Body:       &bufCloser{*bytes.NewBufferString(`{"code":"not_found","message":"foo not found"}`)},
					},
					Error:   nil,
					Request: &Request{},
				}
			}),
			Response{
				&http.Response{
					StatusCode: http.StatusNotFound,
					Header:     http.Header{"Terror": []string{"1"}},
				},
				&terrors.Error{
					Code:        "not_found",
					Message:     "foo not found",
					StackFrames: stack.Stack{},
					Params:      map[string]string{},
				},
				&Request{
					err: nil,
				},
				false,
			},
		},
		{
			"service return non-nil response with empty non-nil error",
			Request{},
			Service(func(req Request) Response {
				return Response{
					Response: &http.Response{
						Header:     http.Header{},
						StatusCode: http.StatusNoContent,
						Body:       &bufCloser{*bytes.NewBufferString(`"message":"no content"`)},
					},
					Error:   errors.New(""),
					Request: &Request{},
				}
			}),
			Response{
				&http.Response{
					Header:     http.Header{},
					StatusCode: http.StatusNoContent,
				},
				errors.New("Response error (204)"),
				&Request{
					err: nil,
				},
				false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRsp := ErrorFilter(tt.req, tt.svc)

			if tt.wantRsp.Response.Body != nil {
				gotResBody, err := ioutil.ReadAll(gotRsp.Response.Body)
				if err != nil {
					t.Fatalf("cannot read gotRsp.Response.Body")
				}
				wantResBody, err := ioutil.ReadAll(tt.wantRsp.Response.Body)
				if err != nil {
					t.Fatalf("cannot read tt.wantRsp.Response.Body")
				}
				assert.Contains(t, string(gotResBody), string(wantResBody))
			}

			assert.Equal(t, tt.wantRsp.Response.Header, gotRsp.Response.Header)
			assert.Equal(t, tt.wantRsp.Response.StatusCode, gotRsp.Response.StatusCode)
			assert.Equal(t, tt.wantRsp.Request, gotRsp.Request)
			assert.Equal(t, tt.wantRsp.hijacked, gotRsp.hijacked)
			assert.EqualValues(t, tt.wantRsp.Error, gotRsp.Error)
		})
	}
}
