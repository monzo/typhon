package typhon

package libs

import (
	"bytes"
	"html/template"
	"log"
	"net/url"
	"strings"

	"github.com/monzo/typhon"
)

type HttpFacade struct {
	Request      typhon.Request
	StatusCode   int
	ResponseBody string
}

func (r HttpFacade) GetFormData() map[string]string {

	requestBody, _ := r.Request.BodyBytes(false)
	body := string(requestBody)

	formData := make(map[string]string)

	// Split the body string by '&'
	pairs := strings.Split(body, "&")

	// Iterate over each key-value pair
	for _, pair := range pairs {
		// Split the pair into key and value
		keyValue := strings.Split(pair, "=")
		// Ensure there are exactly two parts (key and value)
		if len(keyValue) == 2 {
			// Decode the key-value pair and add it to the formData map
			key := keyValue[0]
			value := keyValue[1]
			unescapedPath, err := url.PathUnescape(value)
			if err != nil {
				log.Println(err)
				unescapedPath = value
			}
			formData[key] = unescapedPath
		}
	}

	return formData
}

func (r HttpFacade) ResponseWithJson(statusCode int, responseBody string) typhon.Response {

	instance := typhon.NewResponse(r.Request)
	instance.Writer().Header().Set("Content-Type", "application/json")
	instance.Writer().Write([]byte(responseBody))
	instance.Writer().WriteHeader(statusCode)

	return instance
}

func (r HttpFacade) ResponseWithHtml(statusCode int, responseBody string) typhon.Response {

	instance := typhon.NewResponse(r.Request)
	instance.Writer().Header().Set("Content-Type", "text/html")
	instance.Writer().WriteHeader(statusCode)
	instance.Writer().Write([]byte(responseBody))

	return instance
}

func (r HttpFacade) ResponseWithView(statusCode int, viewPath string, data any) typhon.Response {

	instance := typhon.NewResponse(r.Request)
	instance.Writer().Header().Set("Content-Type", "text/html")
	instance.Writer().WriteHeader(statusCode)
	instance.Writer().Write(r.RenderView(viewPath, data))

	return instance
}

func (v HttpFacade) RenderView(viewPath string, data any) []byte {

	tmpl, err := template.ParseFiles(viewPath)
	ErrCheck(err, true)

	var resultHTML bytes.Buffer
	err = tmpl.Execute(&resultHTML, data)
	ErrCheck(err, true)

	return resultHTML.Bytes()

}

func (v HttpFacade) RenderViewString(viewPath string, data any) string {
	return string(v.RenderView(viewPath, data))
}
