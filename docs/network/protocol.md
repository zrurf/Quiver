# 网络协议

本次选择KCP+Flatbuffers。

## 协议选择
~~无脑推KCP。~~

[KCP](https://github.com/skywind3000/kcp)是一个开源的应用层可靠传输协议，提供了ARQ等机制，且与下层传输协议无关。其还支持选择性重传、快速重传等nb特性，***“能以比 TCP 浪费 10%-20% 的带宽的代价，换取平均延迟降低 30%-40%”***（节选自KCP README.md）。

个人感觉KCP比QUIC、ENet、RakNet等更适合游戏开发，是本人最喜欢的应用层协议之一。很多知名项目，例如原神、网易UU等都在用KCP。KCP也有多种语言的社区绑定，其中Go绑定[kcp-go](https://github.com/xtaci/kcp-go)更是维护积极且受到社区欢迎。因此综合考虑下，使用KCP。