using Newtonsoft.Json;
using System;

namespace Quiver.Api.Model
{
    /// <summary>
    /// 通用 API 响应包装类
    /// </summary>
    /// <typeparam name="T">Payload 数据类型</typeparam>
    public class ApiResponse<T> where T : class
    {
        /// <summary>
        /// 业务状态码，0 表示成功
        /// </summary>
        [JsonProperty("code")]
        public int Code { get; set; }

        /// <summary>
        /// 响应消息
        /// </summary>
        [JsonProperty("msg")]
        public string? Message { get; set; }

        /// <summary>
        /// 状态标识字符串
        /// </summary>
        [JsonProperty("status")]
        public string? Status { get; set; }

        /// <summary>
        /// 服务器时间戳（毫秒）
        /// </summary>
        [JsonProperty("ts")]
        public long Timestamp { get; set; }

        /// <summary>
        /// 响应数据载荷
        /// </summary>
        [JsonProperty("payload")]
        public T? Payload { get; set; }

        /// <summary>
        /// 判断响应是否成功
        /// </summary>
        [JsonIgnore]
        public bool IsSuccess => Code == 0;

        /// <summary>
        /// 创建成功响应
        /// </summary>
        public static ApiResponse<T> Success(T payload, string message = "OK")
        {
            return new ApiResponse<T>
            {
                Code = 0,
                Message = message,
                Status = "OK",
                Timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds(),
                Payload = payload
            };
        }

        /// <summary>
        /// 创建错误响应
        /// </summary>
        public static ApiResponse<T> Error(int code, string message, string status)
        {
            return new ApiResponse<T>
            {
                Code = code,
                Message = message,
                Status = status,
                Timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds(),
                Payload = null
            };
        }
    }

    /// <summary>
    /// 非泛型响应类，用于不关心 Payload 类型的场景
    /// </summary>
    public class ApiResponse : ApiResponse<object>
    {
    }
}
