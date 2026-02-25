// scripts/core/PacketHandler.cs
using game_proto;
using net_proto;
using Godot;
using Google.FlatBuffers;
using System;

public partial class PacketHandler : Node
{
    private GameStateManager _gameState; // 引用游戏状态

    public static bool EncryptionEnabled = true;   // 从配置读取
    public static bool CompressionEnabled = true;

    public override void _Ready()
    {
        NetworkManager.Instance.OnMessageReceived += OnRawMessage;
    }

    private void OnRawMessage(byte[] raw)
    {
        // 前4字节是长度（小端），后面是Flatbuffers数据
        if (raw.Length < 4) return;
        uint len = BitConverter.ToUInt32(raw, 0);
        if (raw.Length - 4 != len) return; // 长度校验

        var fbData = new ByteBuffer(raw, 4);
        var msg = Message.GetRootAsMessage(fbData);

        // 解析头部
        var header = msg.Header;
        if (header.HasValue)
        {
            // 根据消息类型分发
            switch (msg.BodyType)
            {
                case AnyMessage.AuthResponse:
                    HandleAuthResponse(msg.Body<AuthResponse>().Value);
                    break;
                case AnyMessage.JoinRoomResponse:
                    HandleJoinRoomResponse(msg.Body<JoinRoomResponse>().Value);
                    break;
                case AnyMessage.GameData:
                    HandleGameData(msg.Body<GameData>().Value);
                    break;
                case AnyMessage.Heartbeat:
                    // 忽略或回复心跳
                    break;
            }
        }
    }

    private void HandleAuthResponse(AuthResponse resp)
    {
        if (resp.Success)
        {
            GD.Print("认证成功，网关会话ID: " + resp.SessionId);
            // 进入匹配/大厅状态
            GameStateMachine.Instance.ChangeState(GameStateManager.Lobby);
        }
        else
        {
            GD.PrintErr("认证失败: " + resp.ErrorMessage);
        }
    }

    private void HandleJoinRoomResponse(JoinRoomResponse resp)
    {
        if (resp.Success)
        {
            GD.Print($"加入房间成功: {resp.RoomId} 服务器: {resp.GameServerAddr}");
            // 通知游戏状态，开始加载地图等
            GameStateMachine.Instance.ChangeState(GameStateManager.InGame);
        }
        else
        {
            GD.PrintErr("加入房间失败");
        }
    }

    private void HandleGameData(GameData gameData)
    {
        // 这里的数据是已经由网关透传的、可能经过加密压缩的游戏数据包
        byte[] payload = gameData.GetDataBytes().GetValueOrDefault().ToArray();

        // 解密（如果启用）和解压
        if (EncryptionEnabled)
        {
            payload = Crypto.Decrypt(payload); // 需实现
        }
        if (CompressionEnabled)
        {
            // 假设原始大小可从某个地方知道，简单起见先直接解压
            payload = Compression.Decompress(payload, payload.Length * 4); // 粗略估计
        }

        // 解析 GamePacket
        var gamePacket = GamePacket.GetRootAsGamePacket(new ByteBuffer(payload));
        ProcessGamePacket(gamePacket);
    }

    private void ProcessGamePacket(GamePacket packet)
    {
        switch (packet.BodyType)
        {
            case GameMessage.GameStateUpdate:
                var update = packet.Body<GameStateUpdate>().Value;
                _gameState.ApplyUpdate(update);
                break;
                // 其他游戏内消息
        }
    }

    // 发送消息到网关
    public static void SendToGateway(AnyMessage body, uint msgType)
    {
        var builder = new FlatBufferBuilder(256);
        var header = PacketHeader.CreatePacketHeader(builder,
            magic: 0x4B435057, version: 1, flags: 0,
            sessionId: SessionManager.SessionId,
            roomId: SessionManager.CurrentRoomId,
            msgType: (ushort)msgType,
            reserved: 0,
            timestamp: (ulong)DateTime.UtcNow.Ticks);

        // 构建Message
        Message.StartMessage(builder);
        Message.AddHeader(builder, header);
        Message.AddBodyType(builder, body.Type);
        int bodyOffset = body.Pack(builder); // 需要每个body实现Pack扩展方法
        Message.AddBody(builder, bodyOffset);
        int msgOffset = Message.EndMessage(builder);
        builder.Finish(msgOffset.Value);

        var data = builder.SizedByteArray();
        // 添加长度头
        var packet = new byte[data.Length + 4];
        BitConverter.GetBytes((uint)data.Length).CopyTo(packet, 0);
        data.CopyTo(packet, 4);
        NetworkManager.Instance.SendPacket(packet);
    }
}