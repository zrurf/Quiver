using Godot;
using System;

public partial class MainMenu : Control
{
	private Label VersionLabel;
	
	[Export] public PackedScene LoginWidget { get; set; }

	public override void _Ready()
	{
		VersionLabel = GetNode<Label>("%VersionLabel");

		if (VersionLabel != null)
		{
			var appVersion = ProjectSettings.GetSetting("application/config/version");
			var engineVersionInfo = Engine.GetVersionInfo();
			VersionLabel.Text = $"Version: {appVersion}";
		}

		LoginWidget = GD.Load<PackedScene>("res://view/widget/login_widget.tscn");

		ShowLoginWidget();
	}
	
	public void ShowLoginWidget() {
		if (LoginWidget == null)
		{
			GD.PrintErr("LoginWidget not yet loaded.");
			return;
		}

		var widgetInstance = LoginWidget.Instantiate<Control>();

		GetNode<CanvasLayer>("%CanvasLayer").AddChild(widgetInstance);

		CallDeferred(nameof(DeferredCenter), widgetInstance);
	}

	private void DeferredCenter(Control widget)
	{
		if (widget.GetParent() is Control parent)
		{
			widget.Position = (parent.Size - widget.Size) / 2;
		}
	}
}
