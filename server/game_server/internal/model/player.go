package model

import (
	"sync"
)

type Player struct {
	UID             int64
	PosX            float32
	PosY            float32
	VelX            float32
	VelY            float32
	Health          int
	Buffs           []uint8
	Level           int
	Exp             int64
	Coins           int64
	Kills           int64
	Deaths          int64
	PlayTime        int64
	Rating          float64
	RatingDeviation float64
	Volatility      float64
	mu              sync.RWMutex
}

// NewPlayer 创建一个新玩家实例
func NewPlayer(uid int64) *Player {
	return &Player{
		UID:    uid,
		Health: 100,
	}
}

// GetPosition 获取玩家位置和速度
func (p *Player) GetPosition() (x, y, vx, vy float32) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.PosX, p.PosY, p.VelX, p.VelY
}

// SetPosition 设置玩家位置和速度
func (p *Player) SetPosition(x, y, vx, vy float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.PosX = x
	p.PosY = y
	p.VelX = vx
	p.VelY = vy
}

// GetHealth 获取当前生命值
func (p *Player) GetHealth() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Health
}

// SetHealth 设置生命值
func (p *Player) SetHealth(h int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health = h
}

// GetBuffs 返回 Buffs 的副本
func (p *Player) GetBuffs() []uint8 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.Buffs == nil {
		return nil
	}
	buffs := make([]uint8, len(p.Buffs))
	copy(buffs, p.Buffs)
	return buffs
}

// SetBuffs 替换 Buffs 切片
func (p *Player) SetBuffs(buffs []uint8) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Buffs = buffs
}
