// +build ignore
// This file demonstrates G3N with full window rendering
// To compile: go build -tags=noaudio main_window.go
// Note: Requires proper OpenGL setup

package main

import (
	"fmt"
	"runtime"

	"github.com/g3n/engine/animation"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/loader/gltf"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/window"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

type App struct {
	window    window.IWindow
	gs        *gls.GLS
	renderer  *renderer.Renderer
	scene     *core.Node
	camera    *camera.Camera
	animMixer *animation.Mixer
}

func main() {
	app := &App{}

	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// Window hints for OpenGL 3.3 core profile
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	// Create window
	win, err := window.New("desktop", 1024, 768, "G3N GLB Animation Test", false)
	if err != nil {
		panic(err)
	}
	app.window = win

	// Create OpenGL state
	gs, err := gls.New()
	if err != nil {
		panic(err)
	}
	app.gs = gs

	// Create renderer
	rend := renderer.NewRenderer(gs)
	if err := rend.AddDefaultShaders(); err != nil {
		panic(err)
	}
	app.renderer = rend

	// Create scene
	app.scene = core.NewNode()

	// Add lights
	ambLight := light.NewAmbient(&math32.Color{R: 0.6, G: 0.6, B: 0.6}, 1.0)
	app.scene.Add(ambLight)

	dirLight := light.NewDirectional(&math32.Color{R: 1, G: 1, B: 1}, 1.0)
	dirLight.SetPosition(5, 10, 5)
	app.scene.Add(dirLight)

	// Create camera
	aspect := float32(1024) / float32(768)
	app.camera = camera.New(aspect)
	app.camera.SetPosition(3, 3, 3)
	app.camera.LookAt(&math32.Vector3{X: 0, Y: 1, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})

	// Add grid
	grid := graphic.NewGridHelper(10, 1, &math32.Color{R: 0.5, G: 0.5, B: 0.5})
	app.scene.Add(grid)

	// Load GLB model
	fmt.Println("Loading model...")
	g, err := gltf.ParseBin("../model.glb")
	if err != nil {
		fmt.Printf("Error loading GLB: %v\n", err)
		panic(err)
	}

	fmt.Printf("Model loaded!\n")
	fmt.Printf("Animations: %d\n", len(g.Animations))

	// Load scene
	modelScene, err := g.LoadScene(0)
	if err != nil {
		panic(err)
	}
	app.scene.Add(modelScene)

	// Setup animation mixer
	if len(g.Animations) > 0 {
		app.animMixer = animation.NewMixer()

		for i, anim := range g.Animations {
			fmt.Printf("Adding animation %d: %s\n", i, anim.Name)
			action := animation.NewAction(anim)
			action.SetLoop(animation.LoopRepeat)
			action.Play()
			app.animMixer.AddAction(action)
		}

		fmt.Println("Animations configured!")
	}

	// Main loop
	lastTime := glfw.GetTime()
	frame := 0

	for !win.ShouldClose() {
		now := glfw.GetTime()
		deltaTime := float32(now - lastTime)
		lastTime = now
		frame++

		// Update animations
		if app.animMixer != nil {
			app.animMixer.Update(deltaTime)
		}

		// Render
		gs.Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

		if err := rend.Render(app.scene, app.camera); err != nil {
			fmt.Printf("Render error: %v\n", err)
		}

		// Swap and poll
		win.SwapBuffers()
		win.PollEvents()

		// Print status every 60 frames
		if frame%60 == 0 {
			fps := 1.0 / deltaTime
			fmt.Printf("Frame %d | FPS: %.1f | Animations: %d\n",
				frame, fps, len(g.Animations))
		}
	}
}
