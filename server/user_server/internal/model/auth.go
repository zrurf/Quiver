package model

// 注册阶段 1
type RegisterInitRequest struct {
	Username            string `json:"username"`
	RegistrationRequest []byte `json:"registration_request"` // 客户端发送的 RegistrationRequest
}

type RegisterInitResponse struct {
	RegistrationResponse []byte `json:"registration_response"` // 服务端返回的 RegistrationResponse
	ServerPublicKey      []byte `json:"server_public_key"`     // 服务器 AKE 公钥
	CredentialIdentifier []byte `json:"credential_identifier,omitempty"`
}

// 注册阶段 2
type RegisterFinalizeRequest struct {
	Username           string `json:"username"`
	RegistrationRecord []byte `json:"registration_record"` // 客户端计算并发来的完整注册记录
}

type RegisterFinalizeResponse struct {
	OK bool `json:"ok"`
}

// 登录阶段 1
type LoginInitRequest struct {
	Username string `json:"username"`
	KE1      []byte `json:"ke1"` // 客户端 KE1
}

type LoginInitResponse struct {
	KE2 []byte `json:"ke2"` // 服务端 KE2
	MAC []byte `json:"mac"` // 登录阶段 2 验证
}

// 登录阶段 2
type LoginFinalizeRequest struct {
	KE3      []byte `json:"ke3"`      // 客户端 KE3
	MAC      []byte `json:"mac"`      // 登录阶段 1 的 MAC
	Username string `json:"username"` // 用户名
}

type LoginFinalizeResponse struct {
	UID          int64  `json:"uid"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpireAt     int64  `json:"expire_at"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}
