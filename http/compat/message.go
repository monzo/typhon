package httpcompat

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"

	"golang.org/x/net/context"

	"github.com/mondough/mercury"
	"github.com/mondough/terrors"
	"github.com/mondough/typhon/http"
	"github.com/mondough/typhon/message"
)

const legacyIdHeader = "Legacy-Id"

func toHeader(m map[string]string) http.Header {
	h := make(http.Header, len(m))
	for k, v := range m {
		h.Set(k, v)
	}
	return h
}

func fromHeader(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) < 1 {
			continue
		}
		m[k] = v[0]
	}
	return m
}

func old2NewRequest(oldReq message.Request) httpsvc.Request {
	v := httpsvc.Request{
		Context: context.Background(),
		Request: http.Request{
			Method: "POST",
			URL: &url.URL{
				Scheme: "http",
				Host:   oldReq.Service(),
				Path:   oldReq.Endpoint()},
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        toHeader(oldReq.Headers()),
			Host:          oldReq.Service(),
			Body:          ioutil.NopCloser(bytes.NewReader(oldReq.Payload())),
			ContentLength: int64(len(oldReq.Payload()))}}
	v.Header.Set(legacyIdHeader, oldReq.Id())
	return v
}

func new2OldRequest(newReq httpsvc.Request) message.Request {
	req := message.NewRequest()
	req.SetService(newReq.Host)
	req.SetEndpoint(newReq.URL.Path)
	b, _ := ioutil.ReadAll(newReq.Body)
	newReq.Body.Close()
	req.SetPayload(b)
	req.SetHeaders(fromHeader(newReq.Header))
	req.SetId(newReq.Header.Get(legacyIdHeader))
	return req
}

func old2NewResponse(oldRsp message.Response) httpsvc.Response {
	mRsp := mercury.FromTyphonResponse(oldRsp)
	status := http.StatusOK
	if mRsp.IsError() {
		status = terorrToHttp(terrors.Wrap(mRsp.Error(), nil).(*terrors.Error).Code)
	}
	v := httpsvc.Response{
		Error: mRsp.Error(),
		Response: http.Response{
			Status:        http.StatusText(status),
			StatusCode:    status,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        toHeader(mRsp.Headers()),
			Body:          ioutil.NopCloser(bytes.NewReader(mRsp.Payload())),
			ContentLength: int64(len(mRsp.Payload()))}}
	v.Header.Set(legacyIdHeader, oldRsp.Id())
	return v
}

func new2OldResponse(newRsp httpsvc.Response) (message.Response, error) {
	rsp := message.NewResponse()
	rsp.SetHeaders(fromHeader(newRsp.Header))
	defer newRsp.Body.Close()
	b, err := ioutil.ReadAll(newRsp.Body)
	if err != nil {
		return rsp, err
	}
	rsp.SetPayload(b)
	rsp.SetId(newRsp.Header.Get(legacyIdHeader))
	return rsp, newRsp.Error
}
