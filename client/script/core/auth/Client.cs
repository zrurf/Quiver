using Godot;
using Newtonsoft.Json;
using Quiver.Api;
using Quiver.Api.Model;
using System;
using System.Collections.Generic;
using System.Net.Http;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;

namespace Quiver.Auth
{
	/// <summary>
	/// 认证客户端，负责与后端 API 通信并协调 OPAQUE 协议流程
	/// 分离视图逻辑与核心 OPAQUE 实现
	/// </summary>
	public partial class AuthClient : RefCounted
	{
		private readonly System.Net.Http.HttpClient _httpClient;
		private readonly Opaque _opaque;
		private string _baseUrl = "";

		/// <summary>
		/// 当前登录会话状态（登录第一步后设置）
		/// </summary>
		private OpaqueLoginSession? _currentLoginSession;

		/// <summary>
		/// 当前注册会话状态（注册第一步后设置）
		/// </summary>
		private OpaqueRegisterSession? _currentRegisterSession;

		/// <summary>
		/// 当前操作的用户名
		/// </summary>
		private string _currentUsername = "";

		/// <summary>
		/// 当前密码（临时存储以完成多步流程）
		/// </summary>
		private string _currentPassword = "";

		/// <summary>
		/// 构造函数
		/// </summary>
		public AuthClient()
		{
			_httpClient = new System.Net.Http.HttpClient
			{
				Timeout = TimeSpan.FromSeconds(30)
			};
			_opaque = new Opaque();
		}

		/// <summary>
		/// 设置服务器基础 URL
		/// </summary>
		/// <param name="baseUrl">服务器地址，如 http://localhost:8080</param>
		public void SetBaseUrl(string baseUrl)
		{
			_baseUrl = baseUrl.TrimEnd('/');
		}

		#region 注册流程

		/// <summary>
		/// 执行完整的注册流程（三步组合）
		/// </summary>
		/// <param name="username">用户名</param>
		/// <param name="password">密码</param>
		/// <param name="cancellationToken">取消令牌</param>
		/// <returns>注册结果</returns>
		public async Task<OpaqueAuthResult> RegisterAsync(
			string username,
			string password,
			CancellationToken cancellationToken = default)
		{
			if (string.IsNullOrWhiteSpace(username) || string.IsNullOrEmpty(password))
			{
				return new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = "用户名和密码不能为空"
				};
			}

			try
			{
				// 步骤 1: 客户端生成注册请求
				bool startResult = _opaque.StartRegistration(
					password,
					out OpaqueRegisterSession? session,
					out string? registrationRequest);

				if (!startResult || session == null || string.IsNullOrEmpty(registrationRequest))
				{
					return new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = "生成注册请求失败"
					};
				}

				_currentRegisterSession = session;
				_currentUsername = username;
				_currentPassword = password;

				// 步骤 2: 发送注册初始化请求到服务器
				var initRequest = new RegisterInitRequest
				{
					Username = username,
					RegistrationRequest = registrationRequest
				};

				ApiResponse<RegisterInitResponse>? initResponse = await PostAsync<RegisterInitRequest, RegisterInitResponse>(
					"/api/auth/register-init",
					initRequest,
					cancellationToken);

				if (initResponse == null || !initResponse.IsSuccess || initResponse.Payload == null)
				{
					return new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = initResponse?.Message ?? "注册初始化请求失败"
					};
				}

				// 保存服务器公钥用于验证
				_currentRegisterSession.ExpectedServerPublicKey = initResponse.Payload.ServerPublicKey;

				// 步骤 3: 客户端完成注册计算
				bool finishResult = _opaque.FinishRegistration(
					password,
					initResponse.Payload.RegistrationResponse,
					_currentRegisterSession,
					out string? registrationRecord,
					out OpaqueAuthResult? finishAuthResult);

				if (!finishResult || string.IsNullOrEmpty(registrationRecord) || finishAuthResult == null)
				{
					return finishAuthResult ?? new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = "生成注册记录失败"
					};
				}

				// 步骤 4: 发送注册完成请求到服务器
				var finalizeRequest = new RegisterFinalizeRequest
				{
					Username = username,
					RegistrationRecord = registrationRecord
				};

				ApiResponse<RegisterFinalizeResponse>? finalizeResponse =
					await PostAsync<RegisterFinalizeRequest, RegisterFinalizeResponse>(
						"/api/auth/register-finalize",
						finalizeRequest,
						cancellationToken);

				if (finalizeResponse == null || !finalizeResponse.IsSuccess || finalizeResponse.Payload == null)
				{
					return new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = finalizeResponse?.Message ?? "注册完成请求失败"
					};
				}

				// 清理临时数据
				_currentRegisterSession = null;
				_currentUsername = "";
				_currentPassword = "";

				return new OpaqueAuthResult
				{
					Success = true,
					ExportKey = finishAuthResult.ExportKey,
					ServerStaticPublicKey = finishAuthResult.ServerStaticPublicKey
				};
			}
			catch (OperationCanceledException)
			{
				return new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = "注册操作已取消"
				};
			}
			catch (Exception ex)
			{
				GD.PrintErr($"RegisterAsync error: {ex}");
				return new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = $"注册失败: {ex.Message}"
				};
			}
		}

		#endregion

		#region 登录流程

		/// <summary>
		/// 执行完整的登录流程（三步组合）
		/// </summary>
		/// <param name="username">用户名</param>
		/// <param name="password">密码</param>
		/// <param name="cancellationToken">取消令牌</param>
		/// <returns>登录结果，包含 Token、UID、SessionKey 等</returns>
		public async Task<OpaqueAuthResult> LoginAsync(
			string username,
			string password,
			CancellationToken cancellationToken = default)
		{
			if (string.IsNullOrWhiteSpace(username) || string.IsNullOrEmpty(password))
			{
				return new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = "用户名和密码不能为空"
				};
			}

			try
			{
				// 步骤 1: 客户端生成 KE1
				bool startResult = _opaque.StartLogin(
					password,
					out OpaqueLoginSession? session,
					out string? ke1);

				if (!startResult || session == null || string.IsNullOrEmpty(ke1))
				{
					return new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = "生成登录请求失败"
					};
				}

				_currentLoginSession = session;
				_currentUsername = username;
				_currentPassword = password;

				// 步骤 2: 发送登录初始化请求到服务器
				var initRequest = new LoginInitRequest
				{
					Username = username,
					KE1 = ke1
				};

				ApiResponse<LoginInitResponse>? initResponse = await PostAsync<LoginInitRequest, LoginInitResponse>(
					"/api/auth/login-init",
					initRequest,
					cancellationToken);

				if (initResponse == null || !initResponse.IsSuccess || initResponse.Payload == null)
				{
					_currentLoginSession = null;
					return new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = initResponse?.Message ?? "登录初始化请求失败"
					};
				}

				// 步骤 3: 客户端完成登录计算（生成 KE3）
				bool finishResult = _opaque.FinishLogin(
					_currentLoginSession,
					initResponse.Payload.KE2,
					password,
					out string? ke3,
					out OpaqueAuthResult? finishResultObj);

				if (!finishResult || string.IsNullOrEmpty(ke3) || finishResultObj == null || !finishResultObj.Success)
				{
					_currentLoginSession = null;
					return finishResultObj ?? new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = "生成登录响应失败"
					};
				}

				// 步骤 4: 发送登录完成请求到服务器（发送 KE3）
				var finalizeRequest = new LoginFinalizeRequest
				{
					KE3 = ke3
				};

				ApiResponse<LoginFinalizeResponse>? finalizeResponse =
					await PostAsync<LoginFinalizeRequest, LoginFinalizeResponse>(
						"/api/auth/login-finalize",
						finalizeRequest,
						cancellationToken);

				if (finalizeResponse == null || !finalizeResponse.IsSuccess || finalizeResponse.Payload == null)
				{
					_currentLoginSession = null;
					return new OpaqueAuthResult
					{
						Success = false,
						ErrorMessage = finalizeResponse?.Message ?? "登录完成请求失败"
					};
				}

				// 登录成功，组装完整结果
				var result = new OpaqueAuthResult
				{
					Success = true,
					UID = finalizeResponse.Payload.UID,
					Token = finalizeResponse.Payload.Token,
					ExportKey = finishResultObj.ExportKey,
					ServerStaticPublicKey = finishResultObj.ServerStaticPublicKey,
					SessionKey = finishResultObj.SessionKey
				};

				// 清理临时数据
				_currentLoginSession = null;
				_currentUsername = "";
				_currentPassword = "";

				return result;
			}
			catch (OperationCanceledException)
			{
				return new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = "登录操作已取消"
				};
			}
			catch (Exception ex)
			{
				GD.PrintErr($"LoginAsync error: {ex}");
				return new OpaqueAuthResult
				{
					Success = false,
					ErrorMessage = $"登录失败: {ex.Message}"
				};
			}
		}

		#endregion

		#region HTTP 辅助方法

		/// <summary>
		/// 发送 POST 请求
		/// </summary>
		/// <typeparam name="TRequest">请求类型</typeparam>
		/// <typeparam name="TResponse">响应 Payload 类型</typeparam>
		/// <param name="endpoint">API 端点（不含基础 URL）</param>
		/// <param name="requestData">请求数据</param>
		/// <param name="cancellationToken">取消令牌</param>
		/// <returns>API 响应</returns>
		private async Task<ApiResponse<TResponse>?> PostAsync<TRequest, TResponse>(
			string endpoint,
			TRequest requestData,
			CancellationToken cancellationToken) where TResponse : class
		{
			string url = $"{_baseUrl}{endpoint}";
			string jsonContent = JsonConvert.SerializeObject(requestData);

			using var content = new StringContent(jsonContent, Encoding.UTF8, "application/json");

			GD.Print($"HTTP POST {url}: {jsonContent}");

			HttpResponseMessage response = await _httpClient.PostAsync(url, content, cancellationToken);
			string responseBody = await response.Content.ReadAsStringAsync(cancellationToken);

			GD.Print($"HTTP Response {(int)response.StatusCode}: {responseBody}");

			if (!response.IsSuccessStatusCode)
			{
				// 尝试解析错误响应
				try
				{
					var errorResponse = JsonConvert.DeserializeObject<ApiResponse<TResponse>>(responseBody);
					return errorResponse;
				}
				catch
				{
					return new ApiResponse<TResponse>
					{
						Code = (int)response.StatusCode,
						Message = $"HTTP 错误: {response.StatusCode}",
						Status = "ERR_HTTP"
					};
				}
			}

			return JsonConvert.DeserializeObject<ApiResponse<TResponse>>(responseBody);
		}

		#endregion

		/// <summary>
		/// 清理资源
		/// </summary>
		public void Dispose()
		{
			_httpClient?.Dispose();
		}
	}
}
