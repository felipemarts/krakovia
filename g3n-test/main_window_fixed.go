package main

import (
	"fmt"
	"runtime"

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
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// Window hints for OpenGL 3.3 core profile (better Windows compatibility)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Samples, 4) // MSAA for better quality

	// Create window using Init
	if err := window.Init(1024, 768, "G3N GLB Animation Test - Windows"); err != nil {
		panic(err)
	}
	win := window.Get()

	// Create OpenGL state
	gs, err := gls.New()
	if err != nil {
		panic(err)
	}

	// Set clear color (important for Windows!)
	gs.ClearColor(0.2, 0.2, 0.3, 1.0) // Dark blue-gray background

	// Create renderer
	rend := renderer.NewRenderer(gs)
	if err := rend.AddDefaultShaders(); err != nil {
		panic(err)
	}

	// Create scene
	scene := core.NewNode()

	// Add ambient light (softer overall illumination)
	ambLight := light.NewAmbient(&math32.Color{R: 0.5, G: 0.5, B: 0.5}, 1.0)
	scene.Add(ambLight)

	// Add directional light (main light source)
	dirLight := light.NewDirectional(&math32.Color{R: 1, G: 1, B: 1}, 0.8)
	dirLight.SetPosition(5, 10, 5)
	scene.Add(dirLight)

	// Add point light for better model visibility
	pointLight := light.NewPoint(&math32.Color{R: 1, G: 0.9, B: 0.8}, 2.0)
	pointLight.SetPosition(2, 3, 2)
	scene.Add(pointLight)

	// Create camera
	aspect := float32(1024) / float32(768)
	cam := camera.New(aspect)
	cam.SetPosition(0, 1.5, 3) // Adjusted for better view
	cam.LookAt(&math32.Vector3{X: 0, Y: 1, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})

	// Load GLB model
	fmt.Println("Loading model...")
	g, err := gltf.ParseBin("../model.glb")
	if err != nil {
		fmt.Printf("Error loading GLB: %v\n", err)
		panic(err)
	}

	fmt.Printf("Model loaded!\n")
	fmt.Printf("Scenes: %d\n", len(g.Scenes))
	fmt.Printf("Nodes: %d\n", len(g.Nodes))
	fmt.Printf("Animations: %d\n", len(g.Animations))

	// Load scene
	modelScene, err := g.LoadScene(0)
	if err != nil {
		panic(err)
	}
	scene.Add(modelScene)

	fmt.Println("Model added to scene!")
	fmt.Println("Window should now display the model...")
	fmt.Println("Close the window to exit.")

	// Main loop
	lastTime := glfw.GetTime()
	frame := 0

	for !win.ShouldClose() {
		now := glfw.GetTime()
		deltaTime := float32(now - lastTime)
		lastTime = now
		frame++

		// Clear buffers with the set clear color
		gs.Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

		// Enable depth testing
		gs.Enable(gls.DEPTH_TEST)

		// Render scene
		if err := rend.Render(scene, cam); err != nil {
			fmt.Printf("Render error: %v\n", err)
		}

		// Swap buffers and poll events
		win.SwapBuffers()
		win.PollEvents()

		// Print status periodically
		if frame%120 == 0 {
			fps := 1.0 / deltaTime
			fmt.Printf("Frame %d | FPS: %.1f\n", frame, fps)
		}
	}

	fmt.Println("Exiting...")
}
