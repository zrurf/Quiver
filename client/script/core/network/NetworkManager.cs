// scripts/core/NetworkManager.cs
using Godot;
using System;
using System.Net;
using System.Net.Sockets;
using System.Net.Sockets.Kcp;
using System.Threading.Tasks;

public partial class NetworkManager : Node
{
    public static NetworkManager Instance { get; private set; }

    private UdpClient _udp;
    private Kcp<KcpSegment> _kcp;
    private IPEndPoint _remoteEndPoint;
    private bool _isRunning;
    private byte[] _recvBuffer = new byte[4096];

    // 收到完整消息的事件 (已解密/解压后的原始Flatbuffers数据)
    public event Action<byte[]> OnMessageReceived;

    public override void _EnterTree()
    {
        if (Instance != null)
        {
            QueueFree();
            return;
        }
        Instance = this;
    }

    public async Task ConnectAsync(string host, int port)
    {
        _remoteEndPoint = new IPEndPoint(IPAddress.Parse(host), port);
        _udp = new UdpClient();
        _udp.Connect(_remoteEndPoint);

        // 初始化KCP
        _kcp = new Kcp<KcpSegment>(0, (byte[] buffer, int size) =>
        {
            _udp.Send(buffer, size);
        });
        _kcp.NoDelay(1, 20, 2, 1); // 快速模式
        _kcp.WndSize(128, 128);
        _kcp.SetMtu(1400);

        _isRunning = true;
        _ = Task.Run(ReceiveLoop);
        _ = Task.Run(UpdateLoop);
    }

    private void ReceiveLoop()
    {
        while (_isRunning)
        {
            try
            {
                var result = _udp.Receive(ref _remoteEndPoint);
                _kcp.Input(result);
            }
            catch (Exception ex)
            {
                GD.PrintErr($"KCP receive error: {ex}");
            }
        }
    }

    private void UpdateLoop()
    {
        while (_isRunning)
        {
            _kcp.Update(DateTime.UtcNow);
            int recvLen;
            while ((recvLen = _kcp.PeekSize()) > 0)
            {
                var buffer = new byte[recvLen];
                if (_kcp.Recv(buffer) > 0)
                {
                    // 这里收到的是原始应用层数据包（未解压/解密？需按协议处理）
                    // 触发事件，由PacketHandler进一步处理
                    _ = CallDeferred(nameof(OnMessageReceivedDeferred), buffer);
                }
            }
            System.Threading.Thread.Sleep(10); // 避免CPU跑满
        }
    }

    private void OnMessageReceivedDeferred(byte[] data)
    {
        OnMessageReceived?.Invoke(data);
    }

    public void SendPacket(byte[] data)
    {
        if (_kcp != null)
        {
            _kcp.Send(data);
        }
    }

    public override void _ExitTree()
    {
        _isRunning = false;
        _udp?.Close();
    }
}