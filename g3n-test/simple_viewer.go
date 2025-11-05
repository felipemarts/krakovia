package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/g3n/engine/animation"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gls"
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

func main() {
	// Initialize GLFW
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// Set window hints for OpenGL context
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	// Initialize G3N window
	err = window.Init(1024, 768, "G3N Model Viewer")
	if err != nil {
		panic(err)
	}
	win := window.Get()

	// Create OpenGL state
	gs, err := gls.New()
	if err != nil {
		panic(err)
	}

	// Set clear color to dark blue-gray (IMPORTANT for Windows!)
	gs.ClearColor(0.15, 0.15, 0.25, 1.0)

	// Create renderer
	rend := renderer.NewRenderer(gs)
	err = rend.AddDefaultShaders()
	if err != nil {
		panic(err)
	}

	// Create scene
	scene := core.NewNode()

	// Add lights - multiple lights for better visibility
	ambLight := light.NewAmbient(&math32.Color{R: 0.4, G: 0.4, B: 0.4}, 1.0)
	scene.Add(ambLight)

	dirLight1 := light.NewDirectional(&math32.Color{R: 1, G: 1, B: 1}, 0.6)
	dirLight1.SetPosition(3, 5, 3)
	scene.Add(dirLight1)

	dirLight2 := light.NewDirectional(&math32.Color{R: 0.8, G: 0.8, B: 1}, 0.4)
	dirLight2.SetPosition(-3, 3, -3)
	scene.Add(dirLight2)

	// Create camera
	aspect := float32(1024) / float32(768)
	cam := camera.New(aspect)
	cam.SetPosition(0, 1.5, 4)
	cam.LookAt(&math32.Vector3{X: 0, Y: 1, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})

	// Load GLB model
	fmt.Println("Loading GLB model...")
	g, err := gltf.ParseBin("../model.glb")
	if err != nil {
		fmt.Printf("Error loading GLB: %v\n", err)
		panic(err)
	}

	fmt.Printf("✓ Model loaded successfully!\n")
	fmt.Printf("  Scenes: %d\n", len(g.Scenes))
	fmt.Printf("  Nodes: %d\n", len(g.Nodes))
	fmt.Printf("  Meshes: %d\n", len(g.Meshes))
	fmt.Printf("  Animations: %d\n", len(g.Animations))

	// Load scene
	modelScene, err := g.LoadScene(0)
	if err != nil {
		panic(err)
	}

	scene.Add(modelScene)

	fmt.Println("✓ Model added to scene")

	// Setup animations
	var anims []*animation.Animation
	if len(g.Animations) > 0 {
		for i := range g.Animations {
			fmt.Printf("✓ Loading animation %d: '%s'\n", i, g.Animations[i].Name)
			fmt.Printf("  Channels: %d | Samplers: %d\n", len(g.Animations[i].Channels), len(g.Animations[i].Samplers))

			// Load animation using GLTF loader
			anim, err := g.LoadAnimation(i)
			if err != nil {
				fmt.Printf("  ⚠ Error loading animation: %v\n", err)
				continue
			}

			// Configure animation
			anim.SetLoop(true)     // Loop animation
			anim.SetPaused(false)  // Ensure it's playing
			anims = append(anims, anim)
		}

		fmt.Printf("✓ %d animations configured and playing!\n", len(anims))
	} else {
		fmt.Println("⚠ No animations found in model")
	}

	fmt.Println("✓ Rendering...")
	fmt.Println("  Close window to exit")

	// Main rendering loop
	frameCount := 0
	startTime := time.Now()
	lastPrintTime := startTime
	lastUpdateTime := startTime

	for {
		// Check if window should close
		if win.(*window.GlfwWindow).ShouldClose() {
			break
		}

		// Calculate current time and delta
		currentTime := time.Now()
		deltaTime := float32(currentTime.Sub(lastUpdateTime).Seconds())
		lastUpdateTime = currentTime

		// Update animations
		for _, anim := range anims {
			anim.Update(deltaTime)
		}

		// Clear the screen
		gs.Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

		// Enable depth testing
		gs.Enable(gls.DEPTH_TEST)

		// Render the scene
		err := rend.Render(scene, cam)
		if err != nil {
			fmt.Printf("Render error: %v\n", err)
		}

		// Swap buffers
		win.(*window.GlfwWindow).SwapBuffers()

		// Poll events
		glfw.PollEvents()

		frameCount++

		// Print FPS every second
		if currentTime.Sub(lastPrintTime) >= time.Second {
			fps := float64(frameCount) / currentTime.Sub(lastPrintTime).Seconds()
			fmt.Printf("  FPS: %.1f | Frames: %d | Time: %.1fs\n",
				fps, frameCount, currentTime.Sub(startTime).Seconds())
			frameCount = 0
			lastPrintTime = currentTime
		}
	}

	fmt.Println("\nExiting...")
}
