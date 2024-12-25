package scene

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/informeai/gocraft/internal"
)

var (
	PprofPort = flag.String("pprof", "", "http pprof port")

	game  *Game
	scale = 1.0
)

type Game struct {
	win *glfw.Window

	Camera   *Camera
	lx, ly   float64
	vy       float32
	prevtime float64

	blockRender  *BlockRender
	lineRender   *LineRender
	playerRender *PlayerRender

	world   *World
	itemidx int
	item    int
	fps     FPS

	exclusiveMouse bool
	closed         bool
}

func initGL(w, h int) *glfw.Window {
	err := glfw.Init()
	if err != nil {
		log.Fatal(err)
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, gl.TRUE)

	win, err := glfw.CreateWindow(w, h, "gocraft", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	win.MakeContextCurrent()
	err = gl.Init()
	if err != nil {
		log.Fatal(err)
	}
	glfw.SwapInterval(1) // enable vsync
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	return win
}

func NewGame(w, h int) (*Game, error) {
	var err error
	game = &Game{}
	game.item = internal.AvailableItems[0]

	mainthread.Call(func() {
		win := initGL(w, h)
		win.SetMouseButtonCallback(game.onMouseButtonCallback)
		win.SetCursorPosCallback(game.onCursorPosCallback)
		win.SetFramebufferSizeCallback(game.onFrameBufferSizeCallback)
		win.SetKeyCallback(game.onKeyCallback)
		game.win = win
	})
	game.world = NewWorld()
	game.Camera = NewCamera(mgl32.Vec3{0, 16, 0})
	game.blockRender, err = NewBlockRender()
	if err != nil {
		return nil, err
	}
	mainthread.Call(func() {
		game.blockRender.UpdateItem(game.item)
	})
	game.lineRender, err = NewLineRender()
	if err != nil {
		return nil, err
	}
	game.playerRender, err = NewPlayerRender()
	if err != nil {
		return nil, err
	}
	go game.blockRender.UpdateLoop()
	go game.syncPlayerLoop()
	return game, nil
}

func (g *Game) setExclusiveMouse(exclusive bool) {
	if exclusive {
		g.win.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	} else {
		g.win.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	}
	g.exclusiveMouse = exclusive
}

func (g *Game) dirtyBlock(id internal.Vec3) {
	cid := id.Chunkid()
	g.blockRender.DirtyChunk(cid)
	neighbors := []internal.Vec3{id.Left(), id.Right(), id.Front(), id.Back()}
	for _, neighbor := range neighbors {
		chunkid := neighbor.Chunkid()
		if chunkid != cid {
			g.blockRender.DirtyChunk(chunkid)
		}
	}
}

func (g *Game) onMouseButtonCallback(win *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
	if !g.exclusiveMouse {
		g.setExclusiveMouse(true)
		return
	}
	head := internal.NearBlock(g.Camera.Pos())
	foot := head.Down()
	block, prev := g.world.HitTest(g.Camera.Pos(), g.Camera.Front())
	fmt.Printf("BLOCK: %+v\n", block)
	fmt.Printf("PREV: %+v\n", prev)
	fmt.Printf("GAME: %+v\n", g)
	if button == glfw.MouseButton2 && action == glfw.Press {
		if prev != nil && *prev != head && *prev != foot {
			g.world.UpdateBlock(*prev, g.item)
			g.dirtyBlock(*prev)
			go ClientUpdateBlock(*prev, g.item)
		}
	}
	if button == glfw.MouseButton1 && action == glfw.Press {
		if block != nil {
			g.world.UpdateBlock(*block, 0)
			g.dirtyBlock(*block)
			go ClientUpdateBlock(*block, 0)
		}
	}
}

func (g *Game) onFrameBufferSizeCallback(window *glfw.Window, width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}

func (g *Game) onCursorPosCallback(win *glfw.Window, xpos float64, ypos float64) {
	if !g.exclusiveMouse {
		return
	}
	if g.lx == 0 && g.ly == 0 {
		g.lx, g.ly = xpos, ypos
		return
	}
	dx, dy := xpos-g.lx, g.ly-ypos
	g.lx, g.ly = xpos, ypos
	g.Camera.OnAngleChange(float32(dx), float32(dy))
}

func (g *Game) onKeyCallback(win *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action != glfw.Press {
		return
	}
	fmt.Printf("GAME: %+v\n", g)
	offsetKey := 48 //offset key map glfw
	switch key {
	case glfw.KeyTab:
		g.Camera.FlipFlying()
	case glfw.KeySpace:
		block := g.CurrentBlockid()
		if g.world.HasBlock(internal.Vec3{block.X, block.Y - 2, block.Z}) {
			g.vy = 8
		}
	case glfw.Key1, glfw.Key2, glfw.Key3, glfw.Key4, glfw.Key5, glfw.Key6, glfw.Key7, glfw.Key8, glfw.Key9:
		g.item = int(key) - offsetKey
		g.blockRender.UpdateItem(g.item)

	case glfw.KeyE:
		g.itemidx = (1 + g.itemidx) % len(internal.AvailableItems)
		g.item = internal.AvailableItems[g.itemidx]
		g.blockRender.UpdateItem(g.item)

	case glfw.KeyR:
		g.itemidx--
		if g.itemidx < 0 {
			g.itemidx = len(internal.AvailableItems) - 1
		}
		g.item = internal.AvailableItems[g.itemidx]
		g.blockRender.UpdateItem(g.item)
	case glfw.KeyO:
		if !g.Camera.flying {
			g.Camera.FlipFlying()
			g.Camera.OnMoveChange(MoveUp, 8)
		}
	}
}

func (g *Game) handleKeyInput(dt float64) {
	speed := float32(0.1)
	if g.Camera.flying {
		speed = 0.2
	}
	if g.win.GetKey(glfw.KeyEscape) == glfw.Press {
		g.setExclusiveMouse(false)
	}
	if g.win.GetKey(glfw.KeyW) == glfw.Press {
		g.Camera.OnMoveChange(MoveForward, speed)
	}
	if g.win.GetKey(glfw.KeyS) == glfw.Press {
		g.Camera.OnMoveChange(MoveBackward, speed)
	}
	if g.win.GetKey(glfw.KeyA) == glfw.Press {
		g.Camera.OnMoveChange(MoveLeft, speed)
	}
	if g.win.GetKey(glfw.KeyD) == glfw.Press {
		g.Camera.OnMoveChange(MoveRight, speed)
	}
	pos := g.Camera.Pos()
	stop := false
	if !g.Camera.Flying() {
		g.vy -= float32(dt * 20)
		if g.vy < -50 {
			g.vy = -50
		}
		pos = mgl32.Vec3{pos.X(), pos.Y() + g.vy*float32(dt), pos.Z()}
	}

	pos, stop = g.world.Collide(pos)
	if stop {
		g.vy = 0
	}
	g.Camera.SetPos(pos)
}

func (g *Game) CurrentBlockid() internal.Vec3 {
	pos := g.Camera.Pos()
	return internal.NearBlock(pos)
}

func (g *Game) ShouldClose() bool {
	return g.closed
}

func (g *Game) renderStat() {
	g.fps.Update()
	p := g.Camera.Pos()
	cid := internal.NearBlock(p).Chunkid()
	stat := g.blockRender.Stat()
	title := fmt.Sprintf("[%.2f %.2f %.2f] %v [%d/%d %d] %d", p.X(), p.Y(), p.Z(),
		cid, stat.RendingChunks, stat.CacheChunks, stat.Faces, g.fps.Fps())
	g.win.SetTitle(title)
}

func (g *Game) syncPlayerLoop() {
	tick := time.NewTicker(time.Second / 10)
	for range tick.C {
		ClientUpdatePlayerState(g.Camera.State())
	}
}

func (g *Game) Update() {
	mainthread.Call(func() {
		var dt float64
		now := glfw.GetTime()
		dt = now - g.prevtime
		g.prevtime = now
		if dt > 0.02 {
			dt = 0.02
		}

		g.handleKeyInput(dt)

		gl.ClearColor(0.57, 0.71, 0.77, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		g.blockRender.Draw()
		g.lineRender.Draw()
		g.playerRender.Draw()

		g.renderStat()

		g.win.SwapBuffers()
		glfw.PollEvents()
		g.closed = g.win.ShouldClose()
	})
}

type FPS struct {
	lastUpdate time.Time
	cnt        int
	fps        int
}

func (f *FPS) Update() {
	f.cnt++
	now := time.Now()
	p := now.Sub(f.lastUpdate)
	if p >= time.Second {
		f.fps = int(float64(f.cnt) / p.Seconds())
		f.cnt = 0
		f.lastUpdate = now
	}
}

func (f *FPS) Fps() int {
	return f.fps
}
