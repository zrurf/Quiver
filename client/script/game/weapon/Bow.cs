// scripts/game/weapon/BowWeapon.cs
using game_proto;
using Godot;
using Google.FlatBuffers;

public partial class BowWeapon : WeaponBase
{
    [Export] public Texture2D BowSpriteSheet { get; set; }
    private Sprite2D _bowSprite;

    public override void _Ready()
    {
        _bowSprite = new Sprite2D();
        _bowSprite.Texture = BowSpriteSheet;
        _bowSprite.Hframes = 8; // 假设有8个方向或动作
        AddChild(_bowSprite);
    }

    public override void Shoot(Vector2 aimDirection)
    {
        if (_cooldownTimer > 0) return;

        // 根据aimDirection旋转或选择帧
        float angle = aimDirection.Angle();
        int frame = (int)((angle + Mathf.Pi) / (2 * Mathf.Pi) * _bowSprite.Hframes);
        _bowSprite.Frame = frame;

        // 创建抛射物并发送网络消息
        CreateProjectile(aimDirection);
        _cooldownTimer = Cooldown;
    }

    private void CreateProjectile(Vector2 dir)
    {
        // 本地生成视觉效果，实际伤害由服务器判定
        var projScene = GD.Load<PackedScene>("res://scenes/game/Projectile.tscn");
        var proj = projScene.Instantiate<Projectile>();
        proj.Position = GlobalPosition + dir * 20;
        proj.Direction = dir;
        GetTree().CurrentScene.AddChild(proj);

        // 发送射击消息到服务器
        SendShootPacket(dir);
    }

    private void SendShootPacket(Vector2 aim)
    {
        var builder = new FlatBufferBuilder(64);
        PlayerShoot.StartPlayerShoot(builder);
        PlayerShoot.AddWeaponType(builder, 0); // 0=弓
        PlayerShoot.AddAimX(builder, aim.X);
        PlayerShoot.AddAimY(builder, aim.Y);
        PlayerShoot.AddPower(builder, 1.0f);
        var shoot = PlayerShoot.EndPlayerShoot(builder);
        // 包装发送
    }
}