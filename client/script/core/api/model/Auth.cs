using Newtonsoft.Json;
using System;

namespace Quiver.Api.Model
{
    #region 注册阶段模型

    /// <summary>
    /// 注册初始化请求
    /// </summary>
    public class RegisterInitRequest
    {
        /// <summary>
        /// 用户名
        /// </summary>
        [JsonProperty("username")]
        public string Username { get; set; } = string.Empty;

        /// <summary>
        /// 客户端生成的注册请求数据（base64 编码）
        /// </summary>
        [JsonProperty("registration_request")]
        public string RegistrationRequest { get; set; } = string.Empty;
    }

    /// <summary>
    /// 注册初始化响应
    /// </summary>
    public class RegisterInitResponse
    {
        /// <summary>
        /// 服务端返回的注册响应数据（base64 编码）
        /// </summary>
        [JsonProperty("registration_response")]
        public string RegistrationResponse { get; set; } = string.Empty;

        /// <summary>
        /// 服务器 AKE 公钥（base64 编码）
        /// </summary>
        [JsonProperty("server_public_key")]
        public string ServerPublicKey { get; set; } = string.Empty;

        /// <summary>
        /// 凭据标识符（可选，base64 编码）
        /// </summary>
        [JsonProperty("credential_identifier")]
        public string? CredentialIdentifier { get; set; }
    }

    /// <summary>
    /// 注册完成请求
    /// </summary>
    public class RegisterFinalizeRequest
    {
        /// <summary>
        /// 用户名
        /// </summary>
        [JsonProperty("username")]
        public string Username { get; set; } = string.Empty;

        /// <summary>
        /// 客户端计算的完整注册记录（base64 编码）
        /// </summary>
        [JsonProperty("registration_record")]
        public string RegistrationRecord { get; set; } = string.Empty;
    }

    /// <summary>
    /// 注册完成响应
    /// </summary>
    public class RegisterFinalizeResponse
    {
        /// <summary>
        /// 是否成功
        /// </summary>
        [JsonProperty("ok")]
        public bool OK { get; set; }
    }

    #endregion

    #region 登录阶段模型

    /// <summary>
    /// 登录初始化请求
    /// </summary>
    public class LoginInitRequest
    {
        /// <summary>
        /// 用户名
        /// </summary>
        [JsonProperty("username")]
        public string Username { get; set; } = string.Empty;

        /// <summary>
        /// 客户端 KE1 数据（base64 编码）
        /// </summary>
        [JsonProperty("ke1")]
        public string KE1 { get; set; } = string.Empty;
    }

    /// <summary>
    /// 登录初始化响应
    /// </summary>
    public class LoginInitResponse
    {
        /// <summary>
        /// 服务端 KE2 数据（base64 编码）
        /// </summary>
        [JsonProperty("ke2")]
        public string KE2 { get; set; } = string.Empty;
    }

    /// <summary>
    /// 登录完成请求
    /// </summary>
    public class LoginFinalizeRequest
    {
        /// <summary>
        /// 客户端 KE3 数据（base64 编码）
        /// </summary>
        [JsonProperty("ke3")]
        public string KE3 { get; set; } = string.Empty;
    }

    /// <summary>
    /// 登录完成响应
    /// </summary>
    public class LoginFinalizeResponse
    {
        /// <summary>
        /// 用户 ID
        /// </summary>
        [JsonProperty("uid")]
        public long UID { get; set; }

        /// <summary>
        /// OPAQUE 会话令牌
        /// </summary>
        [JsonProperty("token")]
        public string Token { get; set; } = string.Empty;
    }

    #endregion

    #region OPAQUE 客户端内部状态

    /// <summary>
    /// OPAQUE 登录会话状态
    /// </summary>
    public class OpaqueLoginSession
    {
        /// <summary>
        /// 客户端登录状态（需保存以完成后续步骤）
        /// </summary>
        public string? ClientLoginState { get; set; }

        /// <summary>
        /// 预期服务器静态公钥（用于验证）
        /// </summary>
        public string? ExpectedServerPublicKey { get; set; }
    }

    /// <summary>
    /// OPAQUE 注册会话状态
    /// </summary>
    public class OpaqueRegisterSession
    {
        /// <summary>
        /// 客户端注册状态（需保存以完成后续步骤）
        /// </summary>
        public string? ClientRegistrationState { get; set; }

        /// <summary>
        /// 预期服务器静态公钥（用于验证）
        /// </summary>
        public string? ExpectedServerPublicKey { get; set; }
    }

    /// <summary>
    /// OPAQUE 认证结果
    /// </summary>
    public class OpaqueAuthResult
    {
        /// <summary>
        /// 是否成功
        /// </summary>
        public bool Success { get; set; }

        /// <summary>
        /// 错误信息
        /// </summary>
        public string? ErrorMessage { get; set; }

        /// <summary>
        /// 用户 ID（登录成功时）
        /// </summary>
        public long? UID { get; set; }

        /// <summary>
        /// 会话令牌（登录成功时）
        /// </summary>
        public string? Token { get; set; }

        /// <summary>
        /// 导出密钥（可用于派生加密密钥）
        /// </summary>
        public string? ExportKey { get; set; }

        /// <summary>
        /// 会话密钥（登录成功时，可用于后续通信加密）
        /// </summary>
        public string? SessionKey { get; set; }

        /// <summary>
        /// 服务器静态公钥（用于验证服务器身份）
        /// </summary>
        public string? ServerStaticPublicKey { get; set; }
    }

    #endregion
}
