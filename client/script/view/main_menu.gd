extends Control

@onready var version_label: Label = $%VersionLabel
@onready var webview: WebView = $%WebView

func _ready() -> void:
	var app_version = ProjectSettings.get_setting("application/config/version")
	var engine_version_info = Engine.get_version_info()
	
	# 输出版本信息
	if version_label != null:
		version_label.text = "Version: %s" % app_version
	
	# Webview配置
	webview.connect("ipc_message", _on_webview_ipc_msg, FLAG_PROCESS_THREAD_MESSAGES_ALL)
	
func _on_webview_ipc_msg(message) -> void:
	var data = JSON.parse_string(message)
	print("Received webview ipc message: %s" % [message])
	match data.action:
		"login":
			if data.payload.uid and data.payload.access_token and data.payload.refresh_token and data.payload.expire_at:
				Global.uid = data.payload.uid
				Global.access_token = data.payload.access_token
				Global.refresh_token = data.payload.refresh_token
				Global.expire_at = data.payload.expire_at
				await get_tree().create_timer(2).timeout
				NodeUitl.disable_node(webview)
		"set_server":
			if data.payload.addr && data.payload.login_url:
				Global.server = data.payload.addr
				webview.load_url("{0}?app=Gopher%26Quiver&server={1}".format([data.payload.login_url,data.payload.addr]))
		_:
			print("Unknown WebView Action")
