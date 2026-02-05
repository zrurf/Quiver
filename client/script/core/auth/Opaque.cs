#nullable enable

using Godot;
using OPAQUE.Net;
using OPAQUE.Net.Types.Parameters;
using OPAQUE.Net.Types.Results;
using Quiver.Api.Model;
using Quiver.Utils;
using System;
using System.Diagnostics.CodeAnalysis;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

namespace Quiver.Auth
{
	/// <summary>
	/// OPAQUE 协议客户端核心实现
	/// 封装 Vaultic.OPAQUE.Net 库，提供类型安全的 API
	/// </summary>
	public class Opaque
	{
		private readonly OpaqueClient _client;

		/// <summary>
		/// 初始化 OPAQUE 客户端
		/// </summary>
		public Opaque()
		{
			_client = new OpaqueClient();
		}

		#region 注册流程

		/// <summary>
		/// 开始注册流程（第一步）
		/// </summary>
		/// <param name="password">用户密码</param>
		/// <param name="session">输出的注册会话状态</param>
		/// <param name="registrationRequest">输出的注册请求数据（base64）</param>
		/// <returns>是否成功</returns>
		public bool StartRegistration(
			string password,
			[NotNullWhen(true)] out OpaqueRegisterSession? session,
			[NotNullWhen(true)] out string? registrationRequest)
		{
			session = null;
			registrationRequest = null;

			try
			{
				bool result = _client.StartRegistration(password, out StartClientRegistrationResult? resultObj);

				if (!result || resultObj == null)
				{
					return false;
				}

				session = new OpaqueRegisterSession
				{
					ClientRegistrationState = resultObj.ClientRegistrationState
				};
				registrationRequest = Base64UrlConverter.FromBase64Url(resultObj.RegistrationRequest);
				
				return true;
			}
			catch (Exception ex)
			{
				GD.PrintErr($"OPAQUE StartRegistration failed: {ex.Message}");
				return false;
			}
		}

		/// <summary>
		/// 完成注册流程（第三步）
		/// </summary>
		/// <param name="password">用户密码</param>
		/// <param name="registrationResponse">服务端返回的注册响应（base64）</param>
		/// <param name="session">注册会话状态</param>
		/// <param name="registrationRecord">输出的注册记录（base64）</param>
		/// <param name="result">输出的认证结果</param>
		/// <returns>是否成功</returns>
		public bool FinishRegistration(
			string password,
			string registrationResponse,
			OpaqueRegisterSession session,
			[NotNullWhen(true)] out string? registrationRecord,
			[NotNullWhen(true)] out OpaqueAuthResult? result)
		{
			registrationRecord = null;
			result = null;

			if (string.IsNullOrEmpty(session.ClientRegistrationState))
			{
				result = new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = "Invalid registration session state"
				};
				return false;
			}

			try
			{
				bool success = _client.FinishRegistration(
					password,
					Base64UrlConverter.ToBase64Url(registrationResponse),
					session.ClientRegistrationState,
					out FinishClientRegistrationResult? finishResult);

				if (!success || finishResult == null)
				{
					result = new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = "FinishRegistration returned false"
					};
					return false;
				}

				registrationRecord = Base64UrlConverter.FromBase64Url(finishResult.RegistrationRecord);
				result = new OpaqueAuthResult
				{
					Success = true,
					ExportKey = Base64UrlConverter.FromBase64Url(finishResult.ExportKey),
					ServerStaticPublicKey = Base64UrlConverter.FromBase64Url(finishResult.ServerStaicPublicKey)
				};

				// 验证服务器公钥（如果之前保存了预期值）
				if (!string.IsNullOrEmpty(session.ExpectedServerPublicKey))
				{
					if (result.ServerStaticPublicKey != session.ExpectedServerPublicKey)
					{
						
						result.Success = false;
						result.ErrorMessage = "Server public key mismatch - potential MITM attack";
						return false;
					}
				}

				return true;
			}
			catch (Exception ex)
			{
				result = new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = $"Registration finalize failed: {ex.Message}"
				};
				GD.PrintErr($"OPAQUE FinishRegistration failed: {ex.Message}");
				return false;
			}
		}

		#endregion

		#region 登录流程

		/// <summary>
		/// 开始登录流程（第一步）
		/// </summary>
		/// <param name="password">用户密码</param>
		/// <param name="session">输出的登录会话状态</param>
		/// <param name="ke1">输出的 KE1 数据（base64）</param>
		/// <returns>是否成功</returns>
		public bool StartLogin(
			string password,
			[NotNullWhen(true)] out OpaqueLoginSession? session,
			[NotNullWhen(true)] out string? ke1)
		{
			session = null;
			ke1 = null;

			try
			{
				bool result = _client.StartLogin(password, out StartClientLoginResult? resultObj);

				if (!result || resultObj == null)
				{
					return false;
				}

				session = new OpaqueLoginSession
				{
					ClientLoginState = resultObj.ClientLoginState
				};
				ke1 = Base64UrlConverter.FromBase64Url(resultObj.StartLoginRequest);

				return true;
			}
			catch (Exception ex)
			{
				GD.PrintErr($"OPAQUE StartLogin failed: {ex.Message}");
				return false;
			}
		}

		/// <summary>
		/// 完成登录流程（第三步）
		/// </summary>
		/// <param name="session">登录会话状态</param>
		/// <param name="ke2">服务端返回的 KE2 数据（base64）</param>
		/// <param name="password">用户密码</param>
		/// <param name="ke3">输出的 KE3 数据（base64），需发送给服务器完成登录</param>
		/// <param name="result">输出的认证结果</param>
		/// <returns>是否成功</returns>
		public bool FinishLogin(
			OpaqueLoginSession session,
			string ke2,
			string password,
			[NotNullWhen(true)] out string? ke3,
			[NotNullWhen(true)] out OpaqueAuthResult? result)
		{
			ke3 = null;
			result = null;

			if (string.IsNullOrEmpty(session.ClientLoginState))
			{
				result = new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = "Invalid login session state"
				};
				return false;
			}

			try
			{
				bool success = _client.FinishLogin(
					session.ClientLoginState,
					Base64UrlConverter.ToBase64Url(ke2),
					password,
					"quiver", "",
					KSFConfig.Create(KSFConfigType.RfcDraftRecommended),
					out FinishClientLoginResult? finishResult);

				if (!success || finishResult == null)
				{
					result = new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = "FinishLogin returned false"
					};
					return false;
				}

				ke3 = Base64UrlConverter.FromBase64Url(finishResult.FinishLoginRequest);

				result = new OpaqueAuthResult
				{
					Success = true,
					ExportKey = Base64UrlConverter.FromBase64Url(finishResult.ExportKey),
					ServerStaticPublicKey = Base64UrlConverter.FromBase64Url(finishResult.ServerStaticPublicKey),
					// SessionKey 可用于后续通信的加密密钥派生
					SessionKey = Base64UrlConverter.FromBase64Url(finishResult.SessionKey)
				};

				// 验证服务器公钥（如果之前保存了预期值）
				if (!string.IsNullOrEmpty(session.ExpectedServerPublicKey))
				{
					if (result.ServerStaticPublicKey != session.ExpectedServerPublicKey)
					{
						result.Success = false;
						result.ErrorMessage = "Server public key mismatch - potential MITM attack";
						return false;
					}
				}

				return true;
			}
			catch (Exception ex)
			{
				result = new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = $"Login finalize failed: {ex.Message}"
				};
				GD.PrintErr($"OPAQUE FinishLogin failed: {ex.Message}");
				return false;
			}
		}

		#endregion
	}
}
