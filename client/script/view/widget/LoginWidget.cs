#nullable enable

using Godot;
using Newtonsoft.Json;
using Newtonsoft.Json.Serialization;
using Quiver.Api.Model;
using Quiver.Auth;
using System;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

/// <summary>
/// 登录/注册界面控件
/// 负责视图逻辑，所有认证逻辑委托给 AuthClient
/// </summary>
public partial class LoginWidget : Control
{
	// 节点引用（通过唯一名称自动获取）
	[Export] public LineEdit? InputServer { get; set; }
	[Export] public LineEdit? InputUName { get; set; }
	[Export] public LineEdit? InputPwd { get; set; }
	[Export] public Button? BtnLogin { get; set; }
	[Export] public Button? BtnReg { get; set; }
	[Export] public Button? BtnCancel { get; set; }
	[Export] public Label? StatusLabel { get; set; }

	/// <summary>
	/// 认证客户端实例
	/// </summary>
	private AuthClient? _authClient;

	/// <summary>
	/// 当前操作的取消令牌源
	/// </summary>
	private CancellationTokenSource? _cancellationTokenSource;

	/// <summary>
	/// 是否正在处理请求（用于防止重复提交）
	/// </summary>
	private bool _isProcessing = false;

	/// <summary>
	/// 登录/注册成功时触发
	/// </summary>
	[Signal]
	public delegate void AuthSuccessEventHandler(long uid, string token, string? exportKey);

	/// <summary>
	/// 取消或关闭时触发
	/// </summary>
	[Signal]
	public delegate void CancelledEventHandler();

	/// <summary>
	/// 初始化
	/// </summary>
	public override void _Ready()
	{
		// 获取节点引用
		ResolveNodeReferences();

		// 初始化 AuthClient
		_authClient = new AuthClient();

		// 绑定按钮事件
		if (BtnLogin != null)
			BtnLogin.Pressed += OnLoginPressed;

		if (BtnReg != null)
			BtnReg.Pressed += OnRegisterPressed;

		if (BtnCancel != null)
			BtnCancel.Pressed += OnCancelPressed;

		// 设置默认值
		if (InputServer != null && string.IsNullOrEmpty(InputServer.Text))
			InputServer.Text = "http://localhost:80";
	}

	/// <summary>
	/// 清理
	/// </summary>
	public override void _ExitTree()
	{
		_cancellationTokenSource?.Cancel();
		_cancellationTokenSource?.Dispose();
		_authClient?.Dispose();
	}

	/// <summary>
	/// 解决节点引用（运行时获取）
	/// </summary>
	private void ResolveNodeReferences()
	{
		InputServer ??= GetNode<LineEdit>("%InputServer");
		InputUName ??= GetNode<LineEdit>("%InputUName");
		InputPwd ??= GetNode<LineEdit>("%InputPwd");
		BtnLogin ??= GetNode<Button>("%BtnLogin");
		BtnReg ??= GetNode<Button>("%BtnReg");
		BtnCancel ??= GetNode<Button>("%BtnCancel");
		StatusLabel ??= GetNode<Label>("%StatusLabel");
	}

	#region 事件处理

	/// <summary>
	/// 登录按钮点击
	/// </summary>
	private void OnLoginPressed()
	{
		if (_isProcessing) return;

		if (!ValidateInputs(out string server, out string username, out string password))
			return;

		_ = ExecuteAuthAsync(server, username, password, isLogin: true);
	}

	/// <summary>
	/// 注册按钮点击
	/// </summary>
	private void OnRegisterPressed()
	{
		if (_isProcessing) return;

		if (!ValidateInputs(out string server, out string username, out string password))
			return;

		_ = ExecuteAuthAsync(server, username, password, isLogin: false);
	}

	/// <summary>
	/// 取消按钮点击
	/// </summary>
	private void OnCancelPressed()
	{
		if (_isProcessing && _cancellationTokenSource != null)
		{
			_cancellationTokenSource.Cancel();
			SetStatus("正在取消...");
		}
		else
		{
			EmitSignal(SignalName.Cancelled);
		}
	}

	#endregion

	#region 核心逻辑

	/// <summary>
	/// 执行认证操作（登录或注册）
	/// </summary>
	private async Task ExecuteAuthAsync(string server, string username, string password, bool isLogin)
	{
		_isProcessing = true;
		_cancellationTokenSource = new CancellationTokenSource();

		SetButtonsEnabled(false);
		SetStatus(isLogin ? "正在登录..." : "正在注册...");

		try
		{
			_authClient?.SetBaseUrl(server);

			OpaqueAuthResult result = isLogin
				? await _authClient!.LoginAsync(username, password, _cancellationTokenSource.Token)
				: await _authClient!.RegisterAsync(username, password, _cancellationTokenSource.Token);

			if (_cancellationTokenSource.Token.IsCancellationRequested)
				return;

			if (result.Success)
			{
				SetStatus(isLogin ? "登录成功！" : "注册成功！");
				EmitSignal(SignalName.AuthSuccess, result.UID ?? 0, result.Token ?? "", result.ExportKey ?? "");
			}
			else
			{
				SetStatus($"失败: {result.ErrorMessage}");
			}
		}
		catch (OperationCanceledException)
		{
			SetStatus("操作已取消");
		}
		catch (Exception ex)
		{
			GD.PrintErr($"Auth error: {ex}");
			SetStatus($"错误: {ex.Message}");
		}
		finally
		{
			_isProcessing = false;
			_cancellationTokenSource?.Dispose();
			_cancellationTokenSource = null;
			SetButtonsEnabled(true);
		}
	}

	/// <summary>
	/// 验证输入
	/// </summary>
	private bool ValidateInputs(out string server, out string username, out string password)
	{
		server = InputServer?.Text?.Trim() ?? "";
		username = InputUName?.Text?.Trim() ?? "";
		password = InputPwd?.Text ?? "";

		if (string.IsNullOrEmpty(server))
		{
			SetStatus("请输入服务器地址");
			InputServer?.GrabFocus();
			return false;
		}

		if (string.IsNullOrEmpty(username))
		{
			SetStatus("请输入用户名");
			InputUName?.GrabFocus();
			return false;
		}

		if (string.IsNullOrEmpty(password))
		{
			SetStatus("请输入密码");
			InputPwd?.GrabFocus();
			return false;
		}

		return true;
	}

	#endregion

	#region UI 辅助方法

	/// <summary>
	/// 设置状态文本
	/// </summary>
	private void SetStatus(string message)
	{
		if (StatusLabel != null)
		{
			StatusLabel.Text = message;
			// 自动调整颜色
			if (message.Contains("成功"))
				StatusLabel.Modulate = new Color(0.2f, 0.8f, 0.2f);
			else if (message.Contains("失败") || message.Contains("错误"))
				StatusLabel.Modulate = new Color(0.9f, 0.2f, 0.2f);
			else if (message.Contains("取消"))
				StatusLabel.Modulate = new Color(0.8f, 0.6f, 0.2f);
			else
				StatusLabel.Modulate = new Color(1, 1, 1);
		}
		GD.Print($"Status: {message}");
	}

	/// <summary>
	/// 设置按钮启用状态
	/// </summary>
	private void SetButtonsEnabled(bool enabled)
	{
		if (BtnLogin != null)
			BtnLogin.Disabled = !enabled;
		if (BtnReg != null)
			BtnReg.Disabled = !enabled;
	}

	#endregion
}
