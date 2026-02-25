// scripts/game/weapon/WeaponManager.cs
using Godot;
using System.Collections.Generic;

public class WeaponManager
{
    private Player _player;
    private List<WeaponBase> _weapons = new();
    private int _currentWeaponIndex;

    public WeaponManager(Player player)
    {
        _player = player;
        // 加载武器配置（从本地或服务器）
        LoadWeapons();
    }

    private void LoadWeapons()
    {
        var bow = new BowWeapon();
        bow.BowSpriteSheet = GD.Load<Texture2D>("res://assets/bows.png");
        bow.Damage = 10;
        bow.Cooldown = 0.5f;
        _player.AddChild(bow);
        _weapons.Add(bow);
    }

    public void Shoot(Vector2 aimDirection)
    {
        _weapons[_currentWeaponIndex]?.Shoot(aimDirection);
    }
}