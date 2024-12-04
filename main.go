package main

/*
#cgo LDFLAGS: -LC:/Users/root/goprojects/windows-sand/ -lsandflakes
#include <stdlib.h>

typedef struct {
    float x;
    float y;
    float speed;
} Sandflake;

extern Sandflake* initialize_sandflakes(size_t count);
extern void update_sandflakes(Sandflake* sandflakes, size_t count, float delta_time);
extern void free_sandflakes(Sandflake* sandflakes);
extern void push_sand(Sandflake* sandflakes, size_t count, float cursor_x, float cursor_y, int direction);
*/
import "C"
import (
	"io/ioutil"
	"log"
	"runtime"
	"strings"
	"syscall"

	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	setWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	getWindowLongPtr = user32.NewProc("GetWindowLongPtrW")
)

var (
	cursorX, cursorY float32
	pushing          bool
	pushDirection    int32
)

const (
	GWL_EXSTYLE       = -20
	WS_EX_LAYERED     = 0x00080000
	WS_EX_TRANSPARENT = 0x00000020
)

func makeWindowPassthrough(hwnd uintptr) {
	gwlExStyle_ := int32(GWL_EXSTYLE)
	gwlExStyle := uintptr(gwlExStyle_)
	style, _, _ := getWindowLongPtr.Call(hwnd, gwlExStyle)
	setWindowLongPtr.Call(hwnd, gwlExStyle, style|WS_EX_LAYERED|WS_EX_TRANSPARENT)
}

func init() {
	runtime.LockOSThread()
}

func main() {
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 6)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Decorated, glfw.False)
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
	glfw.WindowHint(glfw.Floating, glfw.True)

	window, err := glfw.CreateWindow(1919, 1040, "Testing", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	hwnd := uintptr(unsafe.Pointer(window.GetWin32Window()))
	makeWindowPassthrough(hwnd)

	if err := gl.Init(); err != nil {
		panic(err)
	}

	program := createShaderProgram("vertex.glsl", "fragment.glsl")
	gl.UseProgram(program)
	positions := []float32{
		-0.001, -0.001,
		0.001, -0.001,
		0.001, 0.001,
		-0.001, 0.001,
	}
	var vbo, vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)

	gl.BindVertexArray(vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(positions)*4, gl.Ptr(positions), gl.STATIC_DRAW)

	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, nil)
	gl.EnableVertexAttribArray(0)

	count := 100000
	sandflakes := C.initialize_sandflakes(C.size_t(count))
	defer C.free_sandflakes(sandflakes)
	sandflakeSlice := (*[1 << 30]C.Sandflake)(unsafe.Pointer(sandflakes))[:count:count]

	window.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		// Преобразуем положение мыши в систему координат OpenGL [-1, 1]
		cursorX = float32((xpos/1919.0)*2 - 1)    // Окно шириной 640
		cursorY = float32(-((ypos/1040.0)*2 - 1)) // Окно высотой 480
	})

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		if button == glfw.MouseButtonLeft {
			if action == glfw.Press {
				pushing = true
				pushDirection = 1 // Вправо
			} else if action == glfw.Release {
				pushing = false
			}
		} else if button == glfw.MouseButtonRight {
			if action == glfw.Press {
				pushing = true
				pushDirection = -1 // Влево
			} else if action == glfw.Release {
				pushing = false
			}
		}
	})

	lastTime := glfw.GetTime()
	for !window.ShouldClose() {
		// Do OpenGL stuff.
		currentTime := glfw.GetTime()
		deltaTime := float32(currentTime - lastTime)
		lastTime = currentTime

		log.Printf("x = %v ;y = %v\n", cursorX, cursorY)

		C.update_sandflakes(sandflakes, C.size_t(count), C.float(deltaTime))
		if pushing {
			C.push_sand(sandflakes, C.size_t(count), C.float(cursorX), C.float(cursorY), C.int(pushDirection))
		}

		gl.ClearColor(0.0, 0.0, 0.0, 0.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.BindVertexArray(vao)
		for _, s := range sandflakeSlice {
			gl.Uniform2f(gl.GetUniformLocation(program, gl.Str("sandflakePosition\x00")), float32(s.x), float32(s.y))
			gl.DrawArrays(gl.TRIANGLE_FAN, 0, 4)
		}
		if err := gl.GetError(); err != gl.NO_ERROR {
			log.Printf("OpenGL err: %v\n", err)
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func createShaderProgram(vertexPath, fragmentPath string) uint32 {
	vertexShader := compileShader(vertexPath, gl.VERTEX_SHADER)
	fragmentShader := compileShader(fragmentPath, gl.FRAGMENT_SHADER)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		logL := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(logL))

		log.Fatalf("failed to link program: %v\n", logL)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program
}

func compileShader(path string, shaderType uint32) uint32 {
	source, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read %v: %v", path, err)
	}

	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(string(source) + "\x00")
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		logL := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(logL))

		log.Fatalf("failed to compile %v: %v\n", path, logL)
	}

	return shader
}
