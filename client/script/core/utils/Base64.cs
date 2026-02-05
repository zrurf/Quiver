using System;
using System.Collections.Generic;
using System.Text;

namespace Quiver.Utils
{
    public static class Base64UrlConverter
    {
        /// <summary>
        /// 将标准Base64字符串转换为URL安全的Base64字符串
        /// </summary>
        /// <param name="standardBase64">标准Base64字符串</param>
        /// <returns>URL安全的Base64字符串</returns>
        public static string ToBase64Url(string standardBase64)
        {
            if (string.IsNullOrEmpty(standardBase64))
                return standardBase64;

            // 移除所有填充字符 '='
            string base64Url = standardBase64.TrimEnd('=');

            // 替换特殊字符
            base64Url = base64Url.Replace('+', '-').Replace('/', '_');

            return base64Url;
        }

        /// <summary>
        /// 将字节数组直接编码为URL安全的Base64字符串
        /// </summary>
        /// <param name="data">要编码的字节数组</param>
        /// <returns>URL安全的Base64字符串</returns>
        public static string EncodeToBase64Url(byte[] data)
        {
            if (data == null || data.Length == 0)
                return string.Empty;

            // 先转换为标准Base64
            string standardBase64 = Convert.ToBase64String(data);

            // 再转换为URL Base64
            return ToBase64Url(standardBase64);
        }

        /// <summary>
        /// 将字符串编码为URL安全的Base64字符串
        /// </summary>
        /// <param name="text">要编码的文本</param>
        /// <param name="encoding">使用的编码（默认为UTF-8）</param>
        /// <returns>URL安全的Base64字符串</returns>
        public static string EncodeToBase64Url(string text, Encoding encoding = null)
        {
            if (string.IsNullOrEmpty(text))
                return string.Empty;

            encoding ??= Encoding.UTF8;
            byte[] data = encoding.GetBytes(text);

            return EncodeToBase64Url(data);
        }

        /// <summary>
        /// 将URL安全的Base64字符串转换为标准Base64字符串
        /// </summary>
        /// <param name="base64Url">URL安全的Base64字符串</param>
        /// <returns>标准Base64字符串</returns>
        public static string FromBase64Url(string base64Url)
        {
            if (string.IsNullOrEmpty(base64Url))
                return base64Url;

            // 替换回标准Base64字符
            string base64 = base64Url
                .Replace('-', '+')
                .Replace('_', '/');

            // 补充填充字符
            int padding = base64.Length % 4;
            if (padding > 0)
            {
                base64 += new string('=', 4 - padding);
            }

            return base64;
        }

        /// <summary>
        /// 解码URL安全的Base64字符串为字节数组
        /// </summary>
        /// <param name="base64Url">URL安全的Base64字符串</param>
        /// <returns>解码后的字节数组</returns>
        public static byte[] DecodeFromBase64Url(string base64Url)
        {
            if (string.IsNullOrEmpty(base64Url))
                return Array.Empty<byte>();

            // 先转换为标准Base64
            string standardBase64 = FromBase64Url(base64Url);

            // 解码为字节数组
            return Convert.FromBase64String(standardBase64);
        }

        /// <summary>
        /// 解码URL安全的Base64字符串为文本
        /// </summary>
        /// <param name="base64Url">URL安全的Base64字符串</param>
        /// <param name="encoding">使用的编码（默认为UTF-8）</param>
        /// <returns>解码后的文本</returns>
        public static string DecodeStringFromBase64Url(string base64Url, Encoding encoding = null)
        {
            if (string.IsNullOrEmpty(base64Url))
                return string.Empty;

            byte[] data = DecodeFromBase64Url(base64Url);
            encoding ??= Encoding.UTF8;

            return encoding.GetString(data);
        }

        /// <summary>
        /// 验证是否为有效的URL Base64字符串
        /// </summary>
        /// <param name="input">要验证的字符串</param>
        /// <returns>是否是有效的URL Base64</returns>
        public static bool IsValidBase64Url(string input)
        {
            if (string.IsNullOrEmpty(input))
                return false;

            try
            {
                // 尝试转换为标准Base64并解码
                string standardBase64 = FromBase64Url(input);

                // 检查标准Base64的有效性
                if (standardBase64.Length % 4 != 0)
                    return false;

                // 尝试解码验证
                Convert.FromBase64String(standardBase64);
                return true;
            }
            catch
            {
                return false;
            }
        }

        /// <summary>
        /// 清理Base64字符串（移除空格、换行等无效字符）
        /// </summary>
        /// <param name="base64">Base64字符串</param>
        /// <returns>清理后的Base64字符串</returns>
        public static string CleanBase64(string base64)
        {
            if (string.IsNullOrEmpty(base64))
                return base64;

            // 移除常见无效字符
            return base64
                .Replace(" ", "")
                .Replace("\r", "")
                .Replace("\n", "")
                .Replace("\t", "");
        }
    }
}
