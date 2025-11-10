package game

import rl "github.com/gen2brain/raylib-go/raylib"

// Input é uma interface para abstrair entrada do usuário
// Permite testar lógica do jogo sem dependência do Raylib
type Input interface {
	IsForwardPressed() bool
	IsBackPressed() bool
	IsLeftPressed() bool
	IsRightPressed() bool
	IsJumpPressed() bool
	IsLeftClickPressed() bool
	IsRightClickPressed() bool
	IsFlyTogglePressed() bool
	IsFlyUpPressed() bool
	IsFlyDownPressed() bool
	GetMouseDelta() rl.Vector2
}

// RaylibInput implementa Input usando Raylib real
type RaylibInput struct{}

func (r *RaylibInput) IsForwardPressed() bool {
	return rl.IsKeyDown(rl.KeyW)
}

func (r *RaylibInput) IsBackPressed() bool {
	return rl.IsKeyDown(rl.KeyS)
}

func (r *RaylibInput) IsLeftPressed() bool {
	return rl.IsKeyDown(rl.KeyA)
}

func (r *RaylibInput) IsRightPressed() bool {
	return rl.IsKeyDown(rl.KeyD)
}

func (r *RaylibInput) IsJumpPressed() bool {
	return rl.IsKeyPressed(rl.KeySpace)
}

func (r *RaylibInput) IsLeftClickPressed() bool {
	return rl.IsMouseButtonPressed(rl.MouseLeftButton)
}

func (r *RaylibInput) IsRightClickPressed() bool {
	return rl.IsMouseButtonPressed(rl.MouseRightButton)
}

func (r *RaylibInput) GetMouseDelta() rl.Vector2 {
	return rl.GetMouseDelta()
}

func (r *RaylibInput) IsFlyTogglePressed() bool {
	return rl.IsKeyPressed(rl.KeyP)
}

func (r *RaylibInput) IsFlyUpPressed() bool {
	return rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
}

func (r *RaylibInput) IsFlyDownPressed() bool {
	return rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
}

// SimulatedInput implementa Input para testes
type SimulatedInput struct {
	Forward    bool
	Back       bool
	Left       bool
	Right      bool
	Jump       bool
	LeftClick  bool
	RightClick bool
	FlyToggle  bool
	FlyUp      bool
	FlyDown    bool
	MouseDelta rl.Vector2
}

func (s *SimulatedInput) IsForwardPressed() bool {
	return s.Forward
}

func (s *SimulatedInput) IsBackPressed() bool {
	return s.Back
}

func (s *SimulatedInput) IsLeftPressed() bool {
	return s.Left
}

func (s *SimulatedInput) IsRightPressed() bool {
	return s.Right
}

func (s *SimulatedInput) IsJumpPressed() bool {
	result := s.Jump
	s.Jump = false // IsKeyPressed só retorna true uma vez
	return result
}

func (s *SimulatedInput) IsLeftClickPressed() bool {
	result := s.LeftClick
	s.LeftClick = false
	return result
}

func (s *SimulatedInput) IsRightClickPressed() bool {
	result := s.RightClick
	s.RightClick = false
	return result
}

func (s *SimulatedInput) GetMouseDelta() rl.Vector2 {
	delta := s.MouseDelta
	s.MouseDelta = rl.NewVector2(0, 0) // Reset após leitura
	return delta
}

func (s *SimulatedInput) IsFlyTogglePressed() bool {
	result := s.FlyToggle
	s.FlyToggle = false
	return result
}

func (s *SimulatedInput) IsFlyUpPressed() bool {
	return s.FlyUp
}

func (s *SimulatedInput) IsFlyDownPressed() bool {
	return s.FlyDown
}
