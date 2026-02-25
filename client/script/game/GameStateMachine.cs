using Godot;
using Google.FlatBuffers;
using net_proto;
using System.Collections.Generic;

public enum GameStateManager
{
    Disconnected,
    Connecting,
    Authenticating,
    Lobby,
    InGame,
}

public partial class GameStateMachine : Node
{
    public static GameStateMachine Instance { get; private set; }

    private GameStateManager _currentState = GameStateManager.Disconnected;
    private Dictionary<GameStateManager, IGameState> _states;

    public override void _EnterTree()
    {
        if (Instance != null)
        {
            QueueFree();
            return;
        }
        Instance = this;

        _states = new Dictionary<GameStateManager, IGameState>
        {
            { GameStateManager.Disconnected, new DisconnectedState() },
            { GameStateManager.Connecting, new ConnectingState() },
            { GameStateManager.Authenticating, new AuthenticatingState() },
            { GameStateManager.Lobby, new LobbyState() },
            { GameStateManager.InGame, new InGameState() }
        };
    }

    public void ChangeState(GameStateManager newState)
    {
        if (_states.ContainsKey(_currentState))
            _states[_currentState].Exit();
        _currentState = newState;
        if (_states.ContainsKey(_currentState))
            _states[_currentState].Enter();
    }

    public override void _Process(double delta)
    {
        if (_states.ContainsKey(_currentState))
            _states[_currentState].Update((float)delta);
    }
}

// 状态接口
public interface IGameState
{
    void Enter();
    void Exit();
    void Update(float delta);
}

// 示例：连接状态
public class ConnectingState : IGameState
{
    public async void Enter()
    {
        var global = (GodotObject)GD.Load<CSharpScript>("res://global.gd").New(); // 实际应通过单例访问
        // 从Global获取服务器地址
        string server = (string)global.Get("server");
        await NetworkManager.Instance.ConnectAsync(server.Split(':')[0], 18550);
        // 连接成功后发送认证
        SendAuth();
    }

    private void SendAuth()
    {
        var builder = new FlatBufferBuilder(128);
        var token = builder.CreateString(Global.AccessToken);
        AuthRequest.StartAuthRequest(builder);
        AuthRequest.AddToken(builder, token);
        var req = AuthRequest.EndAuthRequest(builder);
        PacketHandler.SendToGateway(new AnyMessage(req), AnyMessage.AuthRequest);
    }

    public void Exit() { }
    public void Update(float delta) { }
}