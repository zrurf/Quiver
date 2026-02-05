package api

import "github.com/bytedance/sonic"

type ResponseModel struct {
	Code      int    `json:"code"`
	Message   string `json:"msg"`
	Status    string `json:"status"`
	Timestamp int64  `json:"ts"`
	Payload   any    `json:"payload,omitempty"`
}

func (r *ResponseModel) ToJSON() (string, error) {
	return sonic.MarshalString(&r)
}

func NewResponseModelFromJSON(json string) (*ResponseModel, error) {
	var data ResponseModel
	var err = sonic.UnmarshalString(json, &data)
	return &data, err
}

const (
	CodeSuccess     = 0
	CodeServerError = -1
	CodeInvalidBody = 1

	// Auth
	CodeLoggedIn           = 600
	CodeRegisterInitFailed = 601
	CodeLoginFailed        = 602
)

const (
	StatusSuccess        = "OK"
	StatusErrInvalidBody = "ERR_INVALID_REQUEST_BODY"
	StatusErrServer      = "ERR_SERVER_ERROR"
	StatusErrLogin       = "ERR_LOGIN_FAILED"
)
