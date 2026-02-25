// scripts/game/weapon/WeaponBase.cs
using Godot;

public abstract partial class WeaponBase : Node2D
{
    [Export] public int Damage { get; set; }
    [Export] public float Cooldown { get; set; }
    protected float _cooldownTimer;

    public abstract void Shoot(Vector2 aimDirection);
    public override void _Process(double delta)
    {
        if (_cooldownTimer > 0)
            _cooldownTimer -= (float)delta;
    }
}