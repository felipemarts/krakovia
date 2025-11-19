package game

import (
	"math"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	cameraTargetHeight              = 1.0
	cameraTargetOffsetUp            = 1.0
	cameraTargetOffsetRight         = -0.6
	cameraTransitionSpeed           = 8.0
	cameraAutoFirstPersonSwitch     = 0.8
	cameraCollisionProbeStep        = 0.2
	cameraCollisionPadding          = 0.3
	cameraFirstPersonForwardOffset  = 0.05
	cameraFirstPersonBlendThreshold = 0.15
)

// PlayerModel gerencia o modelo 3D e animações do jogador
type PlayerModel struct {
	Model            rl.Model
	Animations       []rl.ModelAnimation
	AnimationCount   int
	CurrentAnimIndex int
	CurrentFrame     int
	IsLoaded         bool
	AnimationNames   map[string]int // Mapa de nome -> índice da animação

	// Blending de animações
	PrevAnimIndex   int     // Animação anterior (para blend)
	PrevFrame       int     // Frame da animação anterior
	BlendFactor     float32 // Fator de blend (0 = prev, 1 = current)
	BlendSpeed      float32 // Velocidade de transição
	IsBlending      bool    // Se está em transição
}

// LoadPlayerModel carrega um modelo GLB com animações
// Baseado no exemplo raylib de importação GLTF
func LoadPlayerModel(modelPath string) *PlayerModel {
	pm := &PlayerModel{
		IsLoaded:       false,
		AnimationNames: make(map[string]int),
		BlendSpeed:     4.0, // Velocidade de transição (maior = mais rápido, ~250ms)
		BlendFactor:    1.0, // Começa sem blend
	}

	// Carregar modelo GLB
	pm.Model = rl.LoadModel(modelPath)
	if pm.Model.MeshCount == 0 {
		return pm
	}

	// Carregar animações do modelo
	pm.Animations = rl.LoadModelAnimations(modelPath)
	pm.AnimationCount = len(pm.Animations)

	// Criar mapa de nomes das animações
	for i := 0; i < pm.AnimationCount; i++ {
		anim := pm.Animations[i]
		name := ""
		for _, c := range anim.Name {
			if c == 0 {
				break
			}
			name += string(c)
		}
		if name != "" {
			pm.AnimationNames[name] = i
		}
	}

	// Inicializar com primeira animação se disponível
	pm.CurrentAnimIndex = 0
	pm.CurrentFrame = 0
	pm.IsLoaded = true

	// Log de todas as animações carregadas
	//pm.LogAllAnimations()

	return pm
}

// LogAllAnimations imprime informações sobre todas as animações do modelo
func (pm *PlayerModel) LogAllAnimations() {
	if !pm.IsLoaded || pm.AnimationCount == 0 {
		println("No animations loaded")
		return
	}

	println("=== Animações do Modelo ===")
	println("Total:", pm.AnimationCount, "animações")
	println("")

	for i := 0; i < pm.AnimationCount; i++ {
		anim := pm.Animations[i]

		// Converter nome para string
		name := ""
		for _, c := range anim.Name {
			if c == 0 {
				break
			}
			name += string(c)
		}
		if name == "" {
			name = "Unnamed"
		}

		println("Animação", i+1, ":", name)
		println("  - Frames:", anim.FrameCount)
		println("  - Bones:", anim.BoneCount)
		println("")
	}
	println("===========================")
}

// UnloadPlayerModel descarrega o modelo e suas animações
func (pm *PlayerModel) Unload() {
	if !pm.IsLoaded {
		return
	}

	// Descarregar animações
	if pm.AnimationCount > 0 && len(pm.Animations) > 0 {
		rl.UnloadModelAnimations(pm.Animations)
	}

	// Descarregar modelo
	rl.UnloadModel(pm.Model)
	pm.IsLoaded = false
}

// UpdateAnimation atualiza a animação do modelo com blending suave
func (pm *PlayerModel) UpdateAnimation() {
	if !pm.IsLoaded || pm.AnimationCount == 0 {
		return
	}

	// Obter animação atual
	anim := pm.Animations[pm.CurrentAnimIndex]

	// Avançar frame da animação atual
	pm.CurrentFrame = (pm.CurrentFrame + 1) % int(anim.FrameCount)

	// Atualizar blend factor
	if pm.IsBlending {
		// Usar delta time fixo de 1/60 para animação consistente
		pm.BlendFactor += pm.BlendSpeed * (1.0 / 60.0)
		if pm.BlendFactor >= 1.0 {
			pm.BlendFactor = 1.0
			pm.IsBlending = false
		}
	}

	// Aplicar animação com blending
	if pm.IsBlending && pm.PrevAnimIndex >= 0 && pm.PrevAnimIndex < pm.AnimationCount {
		// Atualizar frame da animação anterior também
		prevAnim := pm.Animations[pm.PrevAnimIndex]
		pm.PrevFrame = (pm.PrevFrame + 1) % int(prevAnim.FrameCount)

		// Fazer blend entre as duas animações
		pm.updateModelAnimationBlend(prevAnim, int32(pm.PrevFrame), anim, int32(pm.CurrentFrame), pm.BlendFactor)
	} else {
		// Sem blend, usar animação direta
		rl.UpdateModelAnimation(pm.Model, anim, int32(pm.CurrentFrame))
	}
}

// updateModelAnimationBlend faz interpolação entre duas animações
func (pm *PlayerModel) updateModelAnimationBlend(animA rl.ModelAnimation, frameA int32, animB rl.ModelAnimation, frameB int32, blend float32) {
	// Aplicar smoothstep para transição mais suave
	t := blend * blend * (3 - 2*blend)

	// Obter número de bones
	boneCount := int(animA.BoneCount)
	if boneCount == 0 || animA.BoneCount != animB.BoneCount {
		if t < 0.5 {
			rl.UpdateModelAnimation(pm.Model, animA, frameA)
		} else {
			rl.UpdateModelAnimation(pm.Model, animB, frameB)
		}
		return
	}

	// Acessar FramePoses usando unsafe
	framePosesA := unsafe.Slice(animA.FramePoses, animA.FrameCount)
	framePosesB := unsafe.Slice(animB.FramePoses, animB.FrameCount)

	// Obter os bones do frame atual de cada animação
	bonesA := unsafe.Slice(framePosesA[frameA], boneCount)
	bonesB := unsafe.Slice(framePosesB[frameB], boneCount)

	// Salvar valores originais de B antes de modificar
	originalB := make([]rl.Transform, boneCount)
	for i := 0; i < boneCount; i++ {
		originalB[i] = bonesB[i]
	}

	// Interpolar os bones
	for i := 0; i < boneCount; i++ {
		bonesB[i].Translation = rl.Vector3{
			X: bonesA[i].Translation.X*(1-t) + originalB[i].Translation.X*t,
			Y: bonesA[i].Translation.Y*(1-t) + originalB[i].Translation.Y*t,
			Z: bonesA[i].Translation.Z*(1-t) + originalB[i].Translation.Z*t,
		}

		bonesB[i].Rotation = lerpQuaternion(bonesA[i].Rotation, originalB[i].Rotation, t)

		bonesB[i].Scale = rl.Vector3{
			X: bonesA[i].Scale.X*(1-t) + originalB[i].Scale.X*t,
			Y: bonesA[i].Scale.Y*(1-t) + originalB[i].Scale.Y*t,
			Z: bonesA[i].Scale.Z*(1-t) + originalB[i].Scale.Z*t,
		}
	}

	// Aplicar a animação B com os bones interpolados
	rl.UpdateModelAnimation(pm.Model, animB, frameB)

	// Restaurar valores originais de B
	for i := 0; i < boneCount; i++ {
		bonesB[i] = originalB[i]
	}
}

// lerpQuaternion faz interpolação linear normalizada entre quaternions
func lerpQuaternion(q1, q2 rl.Quaternion, t float32) rl.Quaternion {
	// Verificar se os quaternions estão no mesmo hemisfério
	dot := q1.X*q2.X + q1.Y*q2.Y + q1.Z*q2.Z + q1.W*q2.W
	if dot < 0 {
		q2.X = -q2.X
		q2.Y = -q2.Y
		q2.Z = -q2.Z
		q2.W = -q2.W
	}

	// Interpolar linearmente
	result := rl.Quaternion{
		X: q1.X*(1-t) + q2.X*t,
		Y: q1.Y*(1-t) + q2.Y*t,
		Z: q1.Z*(1-t) + q2.Z*t,
		W: q1.W*(1-t) + q2.W*t,
	}

	// Normalizar
	length := float32(math.Sqrt(float64(result.X*result.X + result.Y*result.Y + result.Z*result.Z + result.W*result.W)))
	if length > 0 {
		result.X /= length
		result.Y /= length
		result.Z /= length
		result.W /= length
	}

	return result
}

// SetAnimation define qual animação deve ser reproduzida com transição suave
func (pm *PlayerModel) SetAnimation(index int) {
	if !pm.IsLoaded || index < 0 || index >= pm.AnimationCount {
		return
	}

	if pm.CurrentAnimIndex != index {
		// Salvar animação anterior para blend
		pm.PrevAnimIndex = pm.CurrentAnimIndex
		pm.PrevFrame = pm.CurrentFrame

		// Mudar para nova animação
		pm.CurrentAnimIndex = index
		pm.CurrentFrame = 0

		// Iniciar transição suave
		pm.BlendFactor = 0.0
		pm.IsBlending = true
	}
}

// SetAnimationByName define a animação pelo nome
func (pm *PlayerModel) SetAnimationByName(name string) {
	if !pm.IsLoaded {
		return
	}

	if index, ok := pm.AnimationNames[name]; ok {
		pm.SetAnimation(index)
	}
}

// NextAnimation avança para a próxima animação
func (pm *PlayerModel) NextAnimation() {
	if !pm.IsLoaded || pm.AnimationCount == 0 {
		return
	}

	pm.CurrentAnimIndex = (pm.CurrentAnimIndex + 1) % pm.AnimationCount
	pm.CurrentFrame = 0
}

// PrevAnimation volta para a animação anterior
func (pm *PlayerModel) PrevAnimation() {
	if !pm.IsLoaded || pm.AnimationCount == 0 {
		return
	}

	pm.CurrentAnimIndex--
	if pm.CurrentAnimIndex < 0 {
		pm.CurrentAnimIndex = pm.AnimationCount - 1
	}
	pm.CurrentFrame = 0
}

// GetCurrentAnimationName retorna o nome da animação atual
func (pm *PlayerModel) GetCurrentAnimationName() string {
	if !pm.IsLoaded || pm.AnimationCount == 0 {
		return "No animations"
	}

	anim := pm.Animations[pm.CurrentAnimIndex]
	// Converter o array de caracteres para string
	name := ""
	for _, c := range anim.Name {
		if c == 0 {
			break
		}
		name += string(c)
	}

	if name == "" {
		return "Unnamed"
	}
	return name
}

// GetAnimationInfo retorna informações sobre a animação atual
func (pm *PlayerModel) GetAnimationInfo() string {
	if !pm.IsLoaded || pm.AnimationCount == 0 {
		return "No model loaded"
	}

	name := pm.GetCurrentAnimationName()
	return name
}

// GetAnimationDisplayInfo retorna informações formatadas para exibição na UI
func (p *Player) GetAnimationDisplayInfo() (string, int, int) {
	if p.Model == nil || !p.Model.IsLoaded || p.Model.AnimationCount == 0 {
		return "No animations", 0, 0
	}
	return p.Model.GetCurrentAnimationName(), p.Model.CurrentAnimIndex + 1, p.Model.AnimationCount
}

// Player representa o jogador
type Player struct {
	Position            rl.Vector3
	Velocity            rl.Vector3
	Camera              rl.Camera3D
	Yaw                 float32
	Pitch               float32
	IsOnGround          bool
	GroundedFrames      int // Contador de frames no chão para evitar oscilação
	LookingAtBlock      bool
	TargetBlock         rl.Vector3
	PlaceBlock          rl.Vector3
	Height              float32
	Radius              float32
	CameraDistance      float32
	FirstPerson         bool
	ThirdPersonDistance float32
	FirstPersonDistance float32
	FlyMode             bool
	ShowCollisionBody   bool
	Model               *PlayerModel
	ModelOpacity        float32 // Opacidade do modelo (0.0 = transparente, 1.0 = opaco)
	IsInteracting       bool    // Se está executando animação de interação
	InteractFrames      int     // Frames restantes da animação de interação
}

func NewPlayer(position rl.Vector3) *Player {
	player := &Player{
		Position:            position,
		Velocity:            rl.NewVector3(0, 0, 0),
		Yaw:                 0,
		Pitch:               0.3, // Olhando um pouco para baixo
		Height:              1.8,
		Radius:              0.3,
		CameraDistance:      5.0,
		ThirdPersonDistance: 5.0,
		FirstPersonDistance: 0.35,
		ModelOpacity:        1.0, // Começa opaco
	}

	// Carregar modelo 3D do player
	player.Model = LoadPlayerModel("assets/model.glb")

	// CÃ¢mera em terceira pessoa
	player.Camera = rl.Camera3D{
		Position:   rl.NewVector3(position.X, position.Y+2, position.Z+5),
		Target:     rl.NewVector3(position.X, position.Y+1, position.Z),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       60.0,
		Projection: rl.CameraPerspective,
	}

	return player
}

func (p *Player) Update(dt float32, world *World, input Input) {
	// Atualizar animação do modelo 3D
	if p.Model != nil && p.Model.IsLoaded {
		p.Model.UpdateAnimation()
	}

	// Alternar animações manualmente com setas esquerda/direita (para debug)
	if p.Model != nil && p.Model.IsLoaded {
		if input.IsPrevAnimationPressed() {
			p.Model.PrevAnimation()
		}
		if input.IsNextAnimationPressed() {
			p.Model.NextAnimation()
		}
	}

	// Toggle fly mode com tecla P
	if input.IsFlyTogglePressed() {
		p.FlyMode = !p.FlyMode
		if p.FlyMode {
			// Ao ativar fly mode, zerar velocidade vertical
			p.Velocity.Y = 0
		}
	}

	// Alternar modos de cÃ¢mera com a tecla V
	if input.IsCameraTogglePressed() {
		p.FirstPerson = !p.FirstPerson
	}

	// Toggle visualização do corpo de colisão com tecla K
	if input.IsCollisionTogglePressed() {
		p.ShowCollisionBody = !p.ShowCollisionBody
	}

	// Controle do mouse
	mouseDelta := input.GetMouseDelta()
	sensitivity := float32(0.003)

	p.Yaw -= mouseDelta.X * sensitivity
	p.Pitch -= mouseDelta.Y * sensitivity // Mantém sensação natural em primeira e terceira pessoa

	// Limitar pitch
	if p.Pitch > 1.5 {
		p.Pitch = 1.5
	}
	if p.Pitch < -1.5 {
		p.Pitch = -1.5
	}

	// Calcular direÃ§Ã£o frontal e lateral
	forward := rl.NewVector3(
		float32(math.Sin(float64(p.Yaw))),
		0,
		float32(math.Cos(float64(p.Yaw))),
	)
	right := rl.NewVector3(
		float32(math.Sin(float64(p.Yaw+math.Pi/2))),
		0,
		float32(math.Cos(float64(p.Yaw+math.Pi/2))),
	)

	// Movimento WASD
	speed := float32(15.0)
	moveInput := rl.NewVector3(0, 0, 0)

	if input.IsForwardPressed() {
		moveInput = rl.Vector3Add(moveInput, forward)
	}
	if input.IsBackPressed() {
		moveInput = rl.Vector3Subtract(moveInput, forward)
	}
	if input.IsLeftPressed() {
		moveInput = rl.Vector3Add(moveInput, right)
	}
	if input.IsRightPressed() {
		moveInput = rl.Vector3Subtract(moveInput, right)
	}

	// Normalizar movimento diagonal
	if rl.Vector3Length(moveInput) > 0 {
		moveInput = rl.Vector3Normalize(moveInput)
		moveInput = rl.Vector3Scale(moveInput, speed)
	}

	p.Velocity.X = moveInput.X
	p.Velocity.Z = moveInput.Z

	// LÃ³gica de fÃ­sica diferente baseado no modo fly
	if p.FlyMode {
		// No modo fly: sem gravidade, controle vertical com Shift/Ctrl
		flySpeed := float32(15.0)
		p.Velocity.Y = 0

		if input.IsFlyUpPressed() {
			p.Velocity.Y = flySpeed
		}
		if input.IsFlyDownPressed() {
			p.Velocity.Y = -flySpeed
		}

		// No modo fly, aplicar movimento sem colisÃµes
		p.Position.X += p.Velocity.X * dt
		p.Position.Y += p.Velocity.Y * dt
		p.Position.Z += p.Velocity.Z * dt
	} else {
		// Modo normal: gravidade e colisÃµes ativas
		gravity := float32(-20.0)
		p.Velocity.Y += gravity * dt

		// Pulo
		if input.IsJumpPressed() && p.IsOnGround {
			p.Velocity.Y = 8.0
			p.IsOnGround = false
		}

		// Aplicar velocidade com detecÃ§Ã£o de colisÃ£o
		p.ApplyMovement(dt, world)
	}

	// Atualizar câmera considerando colisões e transições suaves
	p.updateCamera(dt, world)

	// Raycasting para colocar/remover blocos
	p.RaycastBlocks(world)

	// InteraÃ§Ã£o com blocos
	if input.IsLeftClickPressed() && p.LookingAtBlock {
		// Remover bloco
		world.SetBlock(int32(p.TargetBlock.X), int32(p.TargetBlock.Y), int32(p.TargetBlock.Z), BlockAir)
		// Iniciar animação de interação
		p.startInteractAnimation()
	}

	if input.IsRightClickPressed() && p.LookingAtBlock {
		// Colocar bloco - mas verificar se não colide com o jogador
		placePos := rl.NewVector3(
			float32(int32(p.PlaceBlock.X))+0.5,
			float32(int32(p.PlaceBlock.Y)),
			float32(int32(p.PlaceBlock.Z))+0.5,
		)

		// Verificar se o bloco que vai ser colocado não colide com o jogador
		if !p.wouldBlockCollideWithPlayer(placePos) {
			world.SetBlock(int32(p.PlaceBlock.X), int32(p.PlaceBlock.Y), int32(p.PlaceBlock.Z), BlockStone)
			// Iniciar animação de interação
			p.startInteractAnimation()
		}
	}

	// Atualizar animação baseada no estado do jogador
	p.updateAnimationState()
}

// startInteractAnimation inicia a animação de interação
func (p *Player) startInteractAnimation() {
	if p.Model == nil || !p.Model.IsLoaded {
		return
	}

	p.IsInteracting = true
	// Duração da animação em frames (ajustar conforme necessário)
	p.InteractFrames = 30
	p.Model.SetAnimationByName("Interact")
}

// updateAnimationState define a animação baseada no estado atual do jogador
func (p *Player) updateAnimationState() {
	if p.Model == nil || !p.Model.IsLoaded {
		return
	}

	// Se está interagindo, decrementar frames e manter a animação
	if p.IsInteracting {
		p.InteractFrames--
		if p.InteractFrames <= 0 {
			p.IsInteracting = false
		} else {
			// Manter animação de interação
			return
		}
	}

	// Verificar se está se movendo horizontalmente
	isMoving := p.Velocity.X != 0 || p.Velocity.Z != 0

	// Determinar a animação correta baseada no estado
	var animName string

	if p.FlyMode {
		// Modo voo: sempre Swim_Idle_Loop
		animName = "Swim_Idle_Loop"
	} else if !p.IsOnGround {
		// Pulando/caindo: Jump_Loop
		animName = "Jump_Loop"
	} else if isMoving {
		// Andando: Jog_Fwd_Loop
		animName = "Jog_Fwd_Loop"
	} else {
		// Parado: Idle_Loop
		animName = "Idle_Loop"
	}

	// Definir a animação
	p.Model.SetAnimationByName(animName)
}

func (p *Player) updateCamera(dt float32, world *World) {
	desiredDistance := p.ThirdPersonDistance
	if p.FirstPerson {
		desiredDistance = p.FirstPersonDistance
	}

	p.CameraDistance = smoothApproach(p.CameraDistance, desiredDistance, dt, cameraTransitionSpeed)
	if p.CameraDistance < p.FirstPersonDistance {
		p.CameraDistance = p.FirstPersonDistance
	}

	right := rl.NewVector3(
		float32(math.Cos(float64(p.Yaw))),
		0,
		float32(-math.Sin(float64(p.Yaw))),
	)

	head := rl.NewVector3(p.Position.X, p.Position.Y+cameraTargetHeight, p.Position.Z)
	totalRange := p.ThirdPersonDistance - p.FirstPersonDistance
	shoulderBlend := float32(1.0)
	if totalRange > 0 {
		shoulderBlend = clamp01((p.CameraDistance - p.FirstPersonDistance) / totalRange)
	}

	dynamicOffsetRight := cameraTargetOffsetRight * shoulderBlend
	pivot := rl.Vector3Add(head, rl.Vector3Scale(right, dynamicOffsetRight))
	pivot = rl.Vector3Add(pivot, rl.NewVector3(0, cameraTargetOffsetUp, 0))

	forward := rl.NewVector3(
		float32(math.Sin(float64(p.Yaw)))*float32(math.Cos(float64(p.Pitch))),
		float32(math.Sin(float64(p.Pitch))),
		float32(math.Cos(float64(p.Yaw)))*float32(math.Cos(float64(p.Pitch))),
	)
	if rl.Vector3Length(forward) == 0 {
		forward = rl.NewVector3(0, 0, 1)
	} else {
		forward = rl.Vector3Normalize(forward)
	}
	backward := rl.Vector3Scale(forward, -1)

	collisionDistance := p.resolveCameraCollision(world, pivot, backward, p.CameraDistance)

	useFirstPerson := false
	if collisionDistance < cameraAutoFirstPersonSwitch {
		useFirstPerson = true
	} else if p.CameraDistance <= p.FirstPersonDistance+cameraFirstPersonBlendThreshold {
		// Ainda estamos no "túnel" de transição próximo ao jogador, manter primeira pessoa
		useFirstPerson = true
	} else if p.FirstPerson {
		// Deseja primeira pessoa, mas aguardar aproximação suave
		useFirstPerson = false
	}

	var cameraPos rl.Vector3
	var cameraTarget rl.Vector3

	if useFirstPerson {
		viewPivot := rl.Vector3Add(head, rl.NewVector3(0, cameraTargetOffsetUp*0.5, 0))
		cameraPos = rl.Vector3Add(viewPivot, rl.Vector3Scale(forward, cameraFirstPersonForwardOffset))
		cameraTarget = rl.Vector3Add(cameraPos, forward)
	} else {
		cameraPos = rl.Vector3Add(pivot, rl.Vector3Scale(backward, collisionDistance))
		cameraTarget = rl.Vector3Add(pivot, forward)
	}

	p.Camera.Position = cameraPos
	p.Camera.Target = cameraTarget

	// Atualizar opacidade do modelo baseado na distância real da câmera
	// Usar collisionDistance (distância real após colisões) ou CameraDistance
	actualDistance := collisionDistance
	if useFirstPerson {
		actualDistance = 0.0 // Em primeira pessoa, distância é zero
	}

	// Quando está em primeira pessoa ou muito próximo, modelo fica totalmente transparente
	fadeStartDistance := float32(2.0) // Distância em que começa a ficar transparente
	fadeEndDistance := float32(0.8)   // Distância em que fica totalmente transparente

	if actualDistance <= fadeEndDistance {
		// Totalmente transparente em primeira pessoa ou muito próximo
		p.ModelOpacity = 0.0
	} else if actualDistance <= fadeStartDistance {
		// Transição suave de opaco para transparente
		fadeRange := fadeStartDistance - fadeEndDistance
		p.ModelOpacity = (actualDistance - fadeEndDistance) / fadeRange
	} else {
		// Totalmente opaco em terceira pessoa
		p.ModelOpacity = 1.0
	}
}

func (p *Player) resolveCameraCollision(world *World, pivot, backward rl.Vector3, desired float32) float32 {
	if world == nil {
		return desired
	}

	maxDistance := desired
	if maxDistance < p.FirstPersonDistance {
		maxDistance = p.FirstPersonDistance
	}

	steps := int(maxDistance/cameraCollisionProbeStep) + 1
	for i := 0; i <= steps; i++ {
		distance := float32(i) * cameraCollisionProbeStep
		if distance > maxDistance {
			distance = maxDistance
		}

		point := rl.Vector3Add(pivot, rl.Vector3Scale(backward, distance))
		if p.isCameraObstructed(world, point) {
			clipped := distance - cameraCollisionPadding
			if clipped < p.FirstPersonDistance {
				clipped = p.FirstPersonDistance
			}
			if clipped < 0 {
				clipped = 0
			}
			return clipped
		}
	}

	return maxDistance
}

func (p *Player) isCameraObstructed(world *World, point rl.Vector3) bool {
	if world == nil {
		return false
	}

	x := int32(math.Floor(float64(point.X)))
	y := int32(math.Floor(float64(point.Y)))
	z := int32(math.Floor(float64(point.Z)))

	return world.GetBlock(x, y, z) != BlockAir
}

func smoothApproach(current, target, dt, speed float32) float32 {
	if speed <= 0 || dt <= 0 {
		return target
	}

	factor := 1 - float32(math.Exp(float64(-speed*dt)))
	return current + (target-current)*factor
}

func clamp01(value float32) float32 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func (p *Player) RenderPlayer() {
	// Renderizar modelo 3D se disponível e visível
	if p.Model != nil && p.Model.IsLoaded && p.ModelOpacity > 0.0 {
		// Posição do modelo (centralizado na posição do player)
		modelPos := rl.NewVector3(p.Position.X, p.Position.Y, p.Position.Z)

		// Escala do modelo (ajustar conforme necessário)
		scale := float32(1.0)

		// Criar cor com opacidade baseada na distância da câmera
		alpha := uint8(p.ModelOpacity * 255.0)
		tintColor := rl.Color{R: 255, G: 255, B: 255, A: alpha}

		// Renderizar o modelo com animação e transparência
		rl.DrawModel(p.Model.Model, modelPos, scale, tintColor)
	}

	// Renderizar corpo de colisão se ativado
	if p.ShowCollisionBody {
		base := rl.NewVector3(p.Position.X, p.Position.Y, p.Position.Z)
		top := rl.NewVector3(p.Position.X, p.Position.Y+p.Height, p.Position.Z)

		fillColor := rl.Color{R: 255, G: 229, B: 153, A: 80}
		wireColor := rl.Color{R: 255, G: 140, B: 0, A: 255}

		rl.DrawCylinderEx(base, top, p.Radius, p.Radius, 20, fillColor)
		rl.DrawCylinderWiresEx(base, top, p.Radius, p.Radius, 12, wireColor)
	}
}

func (p *Player) ApplyMovement(dt float32, world *World) {
	// Limitar delta time para evitar tunneling em caso de lag
	// Subdividir movimentos grandes em steps menores
	maxDt := float32(0.016) // ~60 FPS
	remainingDt := dt

	for remainingDt > 0 {
		stepDt := remainingDt
		if stepDt > maxDt {
			stepDt = maxDt
		}
		remainingDt -= stepDt

		// Movimento horizontal (X)
		newPosX := p.Position
		newPosX.X += p.Velocity.X * stepDt
		if !p.CheckCollision(newPosX, world) {
			p.Position.X = newPosX.X
		}

		// Movimento horizontal (Z)
		newPosZ := p.Position
		newPosZ.Z += p.Velocity.Z * stepDt
		if !p.CheckCollision(newPosZ, world) {
			p.Position.Z = newPosZ.Z
		}

		// Movimento vertical (Y)
		newPosY := p.Position
		newPosY.Y += p.Velocity.Y * stepDt

		if !p.CheckCollision(newPosY, world) {
			p.Position.Y = newPosY.Y
		} else {
			if p.Velocity.Y < 0 {
				// Colidiu com o chão
				p.Velocity.Y = 0
			} else if p.Velocity.Y > 0 {
				// Colidiu com o teto
				p.Velocity.Y = 0
			}
		}
	}

	// Verificação de chão: sempre checar se há bloco abaixo
	checkBelowPos := p.Position
	checkBelowPos.Y -= 0.1 // Verificar ligeiramente abaixo

	wasOnGround := p.IsOnGround
	p.IsOnGround = p.CheckCollision(checkBelowPos, world)

	// Se acabou de pousar, resetar o buffer
	if p.IsOnGround && !wasOnGround {
		p.GroundedFrames = 10 // Buffer de frames após pousar
	}

	// Se está no chão, manter o buffer
	if p.IsOnGround {
		p.GroundedFrames = 10
	} else if p.GroundedFrames > 0 {
		// Coyote time: ainda considera no chão por alguns frames
		p.GroundedFrames--
		p.IsOnGround = true
	}
}

// wouldBlockCollideWithPlayer verifica se um bloco na posição dada colidiria com o jogador
func (p *Player) wouldBlockCollideWithPlayer(blockPos rl.Vector3) bool {
	// blockPos é o centro do bloco (x+0.5, y, z+0.5)
	// Verificar colisão do cilindro do jogador com o bloco

	// Distância horizontal do centro do jogador ao centro do bloco
	dx := p.Position.X - blockPos.X
	dz := p.Position.Z - blockPos.Z
	distSq := dx*dx + dz*dz

	// Colisão horizontal (cilíndrica)
	maxDist := p.Radius + 0.5
	if distSq >= maxDist*maxDist {
		return false // Muito longe horizontalmente
	}

	// Colisão vertical
	// O bloco ocupa de blockPos.Y até blockPos.Y+1
	// O jogador ocupa de p.Position.Y até p.Position.Y+p.Height
	blockBottom := blockPos.Y
	blockTop := blockPos.Y + 1.0
	playerBottom := p.Position.Y
	playerTop := p.Position.Y + p.Height

	// Verificar se há sobreposição vertical
	if playerTop <= blockBottom || playerBottom >= blockTop {
		return false // Sem sobreposição vertical
	}

	return true // Colide!
}

func (p *Player) CheckCollision(newPos rl.Vector3, world *World) bool {
	// Verificar colisÃ£o cilÃ­ndrica apropriada
	minX := int32(math.Floor(float64(newPos.X - p.Radius)))
	maxX := int32(math.Floor(float64(newPos.X + p.Radius)))
	minY := int32(math.Floor(float64(newPos.Y)))
	maxY := int32(math.Floor(float64(newPos.Y + p.Height)))
	minZ := int32(math.Floor(float64(newPos.Z - p.Radius)))
	maxZ := int32(math.Floor(float64(newPos.Z + p.Radius)))

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				blockType := world.GetBlock(x, y, z)
				if blockType != BlockAir {
					// OtimizaÃ§Ã£o: ignorar colisÃ£o com blocos completamente ocultos
					// (eles nÃ£o podem ser alcanÃ§ados pelo jogador)
					if world.IsBlockHidden(x, y, z) {
						continue
					}

					// Verificar se realmente colide com o cilindro do jogador
					// Centro do bloco
					blockCenterX := float32(x) + 0.5
					blockCenterZ := float32(z) + 0.5

					// DistÃ¢ncia horizontal do centro do jogador ao centro do bloco
					dx := newPos.X - blockCenterX
					dz := newPos.Z - blockCenterZ
					distSq := dx*dx + dz*dz

					// ColisÃ£o cilÃ­ndrica: verificar se a distÃ¢ncia Ã© menor que a soma dos raios
					// (raio do jogador + raio do bloco que Ã© 0.5)
					maxDist := p.Radius + 0.5
					if distSq < maxDist*maxDist {
						return true
					}
				}
			}
		}
	}

	return false
}

func (p *Player) RaycastBlocks(world *World) {
	// Raycast diretamente da cÃ¢mera na direÃ§Ã£o que ela estÃ¡ apontando
	// Isso garante que o raycast sempre acerte onde o crosshair aponta
	rayOrigin := p.Camera.Position
	rayDir := rl.Vector3Normalize(rl.Vector3Subtract(p.Camera.Target, p.Camera.Position))

	maxDistance := float32(10.0)
	p.LookingAtBlock = false

	// PosiÃ§Ã£o inicial do voxel
	voxelX := int32(math.Floor(float64(rayOrigin.X)))
	voxelY := int32(math.Floor(float64(rayOrigin.Y)))
	voxelZ := int32(math.Floor(float64(rayOrigin.Z)))

	// DireÃ§Ã£o do passo (1 ou -1)
	stepX := int32(1)
	if rayDir.X < 0 {
		stepX = -1
	}
	stepY := int32(1)
	if rayDir.Y < 0 {
		stepY = -1
	}
	stepZ := int32(1)
	if rayDir.Z < 0 {
		stepZ = -1
	}

	// Calcular tMax e tDelta
	var tMaxX, tMaxY, tMaxZ float32
	var tDeltaX, tDeltaY, tDeltaZ float32

	if rayDir.X != 0 {
		if rayDir.X > 0 {
			tMaxX = (float32(voxelX+1) - rayOrigin.X) / rayDir.X
		} else {
			tMaxX = (float32(voxelX) - rayOrigin.X) / rayDir.X
		}
		tDeltaX = float32(math.Abs(float64(1.0 / rayDir.X)))
	} else {
		tMaxX = float32(math.MaxFloat32)
		tDeltaX = float32(math.MaxFloat32)
	}

	if rayDir.Y != 0 {
		if rayDir.Y > 0 {
			tMaxY = (float32(voxelY+1) - rayOrigin.Y) / rayDir.Y
		} else {
			tMaxY = (float32(voxelY) - rayOrigin.Y) / rayDir.Y
		}
		tDeltaY = float32(math.Abs(float64(1.0 / rayDir.Y)))
	} else {
		tMaxY = float32(math.MaxFloat32)
		tDeltaY = float32(math.MaxFloat32)
	}

	if rayDir.Z != 0 {
		if rayDir.Z > 0 {
			tMaxZ = (float32(voxelZ+1) - rayOrigin.Z) / rayDir.Z
		} else {
			tMaxZ = (float32(voxelZ) - rayOrigin.Z) / rayDir.Z
		}
		tDeltaZ = float32(math.Abs(float64(1.0 / rayDir.Z)))
	} else {
		tMaxZ = float32(math.MaxFloat32)
		tDeltaZ = float32(math.MaxFloat32)
	}

	// Armazenar voxel anterior para colocaÃ§Ã£o de blocos
	prevVoxelX, prevVoxelY, prevVoxelZ := voxelX, voxelY, voxelZ

	// DDA traversal
	for t := float32(0); t < maxDistance; {
		// Verificar se o voxel atual contÃ©m um bloco
		if world.GetBlock(voxelX, voxelY, voxelZ) != BlockAir {
			p.LookingAtBlock = true
			p.TargetBlock = rl.NewVector3(float32(voxelX), float32(voxelY), float32(voxelZ))
			p.PlaceBlock = rl.NewVector3(float32(prevVoxelX), float32(prevVoxelY), float32(prevVoxelZ))
			return
		}

		// Armazenar voxel atual antes de avanÃ§ar
		prevVoxelX, prevVoxelY, prevVoxelZ = voxelX, voxelY, voxelZ

		// AvanÃ§ar para o prÃ³ximo voxel
		if tMaxX < tMaxY {
			if tMaxX < tMaxZ {
				voxelX += stepX
				t = tMaxX
				tMaxX += tDeltaX
			} else {
				voxelZ += stepZ
				t = tMaxZ
				tMaxZ += tDeltaZ
			}
		} else {
			if tMaxY < tMaxZ {
				voxelY += stepY
				t = tMaxY
				tMaxY += tDeltaY
			} else {
				voxelZ += stepZ
				t = tMaxZ
				tMaxZ += tDeltaZ
			}
		}
	}
}
