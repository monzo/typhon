package transport

type UppercaseRequest struct {
	S string `json:"s"`
}

type CountRequest struct {
	S string `json:"s"`
}

type UppercaseResponse struct {
	Value string `json:"value"`
}

type CountResponse struct {
	Value int `json:"value"`
}
