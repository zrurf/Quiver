// scenes/game/Game.cs
using game_proto;
using Godot;
using System;

public partial class Game : Node2D
{
    private PackedScene _playerScene = GD.Load<PackedScene>("res://actor/player.tscn");
    private GameStateManager _state;

    public override void _Ready()
    {
        // 监听状态机进入InGame
        GameStateMachine.Instance.ChangeState(GameStateManager.InGame);
        // 请求加入房间（已通过网关完成）
    }

    public void OnGameStateUpdate(GameStateUpdate update)
    {
        // 更新其他玩家
        for (int i = 0; i < update.PlayersLength; i++)
        {
            var pState = update.Players(i).Value;
            uint uid = pState.Uid;
            if (uid == Global.Uid) continue; // 自己

            Node2D playerNode;
            if (!HasNode(uid.ToString()))
            {
                playerNode = _playerScene.Instantiate<Node2D>();
                playerNode.Name = uid.ToString();
                AddChild(playerNode);
            }
            else
            {
                playerNode = GetNode<Node2D>(uid.ToString());
            }
            // 更新位置等
            if (playerNode is Player player)
            {
                player.ApplyServerState(
                    new Vector2(pState.PosX, pState.PosY),
                    Vector2.Zero, // 速度可能需要单独字段
                    false
                );
            }
        }

        // 更新抛射物
        for (int i = 0; i < update.ProjectilesLength; i++)
        {
            var projState = update.Projectiles(i).Value;
            // 创建或更新抛射物
        }
    }
}