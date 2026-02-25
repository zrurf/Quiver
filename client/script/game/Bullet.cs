using Godot;

public partial class Bullet : Sprite2D
{
	public ulong Id;
	public ulong OwnerUid;
	public Vector2 Position;
	public Vector2 Velocity;
	public byte Type;

	public Bullet(game_proto.ProjectileState state)
	{
		Id = state.Id;
		OwnerUid = state.OwnerUid;
		Position = new Vector2(state.PosX, state.PosY);
		Velocity = new Vector2(state.VelX, state.VelY);
		Type = state.ProjType;

		Texture = GD.Load<Texture2D>("res://assets/sprite/weapon/Bow_Spritesheet.png");
		Hframes = 15; // 假设有8帧
		Vframes = 15;
		Frame = Type; // 用子弹类型索引帧
	}

	public override void _Process(double delta)
	{
		// 可以简单插值，但最好由服务器驱动位置，这里我们直接使用服务器下发的精确位置
		// 如果要做平滑，可存储目标位置进行插值
	}
}
