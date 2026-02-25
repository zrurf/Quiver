// scripts/game/player/Player.cs
using game_proto;
using Godot;
using Google.FlatBuffers;
using System;

public partial class Player : CharacterBody2D
{
    [Export] public int PlayerId { get; set; } // 网络ID

    private AnimationPlayer _animationPlayer;
    private Sprite2D _sprite;
    private WeaponManager _weaponManager;

    // 同步状态
    private Vector2 _serverPosition;
    private Vector2 _serverVelocity;
    private bool _serverFlipH;

    public override void _Ready()
    {
        _animationPlayer = GetNode<AnimationPlayer>("%AnimationPlayer");
        _sprite = GetNode<Sprite2D>("%Sprite2D");
        _weaponManager = new WeaponManager(this); // 武器系统

        if (PlayerId == Global.Uid) // 本地玩家
        {
            SetProcess(true);
            SetPhysicsProcess(true);
        }
        else // 其他玩家，只接收更新
        {
            SetProcess(false);
            SetPhysicsProcess(false);
        }
    }

    public override void _Process(double delta)
    {
        // 本地玩家：处理输入并发送到服务器
        HandleInput();
    }

    public override void _PhysicsProcess(double delta)
    {
        if (PlayerId == Global.Uid)
        {
            // 本地玩家：应用本地移动（预测），同时将输入发送给服务器
            MoveAndSlide();
            // 发送移动消息给服务器
            SendMovePacket();
        }
        else
        {
            // 其他玩家：插值到服务器位置
            Position = Position.Lerp(_serverPosition, 0.5f);
            Velocity = _serverVelocity;
            _sprite.FlipH = _serverFlipH;
        }
    }

    private void HandleInput()
    {
        Vector2 inputDir = Input.GetVector("Left", "Right", "Up", "Down");
        Velocity = inputDir * 200f; // 速度常量

        // 更新动画
        if (inputDir != Vector2.Zero)
        {
            _animationPlayer.Play("run");
            _sprite.FlipH = inputDir.X >= 0;
        }
        else
        {
            _animationPlayer.Play("idle_anime");
        }
    }

    private void SendMovePacket()
    {
        var builder = new FlatBufferBuilder(64);
        PlayerMove.StartPlayerMove(builder);
        PlayerMove.AddPosX(builder, Position.X);
        PlayerMove.AddPosY(builder, Position.Y);
        PlayerMove.AddVelX(builder, Velocity.X);
        PlayerMove.AddVelY(builder, Velocity.Y);
        PlayerMove.AddTimestamp(builder, (ulong)Time.GetTicksMsec());
        var move = PlayerMove.EndPlayerMove(builder);
        // 包装成GamePacket发送
        SendGamePacket(GameMessage.PlayerMove, move);
    }

    public void ApplyServerState(Vector2 pos, Vector2 vel, bool flipH)
    {
        _serverPosition = pos;
        _serverVelocity = vel;
        _serverFlipH = flipH;
    }
}