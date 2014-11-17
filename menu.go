package menu

import (
	gltext "github.com/4ydx/gltext"
	glfw "github.com/go-gl/glfw3"
	"github.com/go-gl/glow/gl-core/3.3/gl"
	"github.com/go-gl/mathgl/mgl32"
	"os"
)

type Point struct {
	X, Y float32
}

var vertexShaderSource string = `
#version 330

uniform mat4 matrix;

in vec4 position;

void main() {
  gl_Position = matrix * position;
}
` + "\x00"

var fragmentShaderSource string = `
#version 330

out vec4 fragment_color;

void main() {
  fragment_color = vec4(1,1,1,1);
}
` + "\x00"

type Menu struct {
	// options
	Visible      bool
	ShowOn       glfw.Key
	Height       float32
	Width        float32
	IsAutoCenter bool
	lowerLeft    Point

	// interactive objects
	Font          *gltext.Font
	Labels        []*Label
	TextScaleRate float32 // increment during a scale operation

	// opengl oriented
	WindowWidth   float32
	WindowHeight  float32
	program       uint32 // shader program
	glMatrix      int32  // ortho matrix
	position      uint32 // index location
	vao           uint32
	vbo           uint32
	ebo           uint32
	ortho         mgl32.Mat4
	vboData       []float32
	vboIndexCount int
	eboData       []int32
	eboIndexCount int
}

func (menu *Menu) AddLabel(label *Label, str string) {
	label.Load(menu, menu.Font)
	label.Text.SetString(str)
	label.Text.SetScale(1)
	label.Text.SetPosition(0, 0)
	label.Text.SetColor(0, 0, 0, 1)
	menu.Labels = append(menu.Labels, label)
}

func (menu *Menu) Toggle() {
	menu.Visible = !menu.Visible
	for i := range menu.Labels {
		menu.Labels[i].Reset()
	}
}

func (menu *Menu) Load(width float32, height float32, scale int32) (err error) {
	glfloat_size := 4
	glint_size := 4

	menu.Visible = false
	menu.ShowOn = glfw.KeyM
	menu.Width = width
	menu.Height = height

	// load font
	fd, err := os.Open("font/luximr.ttf")
	if err != nil {
		return
	}
	defer fd.Close()

	menu.Font, err = gltext.LoadTruetype(fd, scale, 32, 127)
	if err != nil {
		return
	}

	// 2DO: make this time dependent rather than fps dependent
	menu.TextScaleRate = 0.01

	// create shader program and define attributes and uniforms
	menu.program, err = gltext.NewProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		return
	}
	menu.glMatrix = gl.GetUniformLocation(menu.program, gl.Str("matrix\x00"))
	menu.position = uint32(gl.GetAttribLocation(menu.program, gl.Str("position\x00")))

	gl.GenVertexArrays(1, &menu.vao)
	gl.GenBuffers(1, &menu.vbo)
	gl.GenBuffers(1, &menu.ebo)

	// vao
	gl.BindVertexArray(menu.vao)

	// vbo
	// specify the buffer for which the VertexAttribPointer calls apply
	gl.BindBuffer(gl.ARRAY_BUFFER, menu.vbo)

	gl.EnableVertexAttribArray(menu.position)
	gl.VertexAttribPointer(
		menu.position,
		2,
		gl.FLOAT,
		false,
		0, // no stride... yet
		gl.PtrOffset(0),
	)

	// ebo
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, menu.ebo)

	// i am guessing that order is important here
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)

	// ebo, vbo data
	menu.vboIndexCount = 4 * 2 // four indices (2 points per index)
	menu.eboIndexCount = 6     // 6 triangle indices for a quad
	menu.vboData = make([]float32, menu.vboIndexCount, menu.vboIndexCount)
	menu.eboData = make([]int32, menu.eboIndexCount, menu.eboIndexCount)
	menu.lowerLeft = menu.findCenter()
	menu.makeBufferData()

	// setup context
	gl.BindVertexArray(menu.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, menu.vbo)
	gl.BufferData(
		gl.ARRAY_BUFFER, glfloat_size*menu.vboIndexCount, gl.Ptr(menu.vboData), gl.DYNAMIC_DRAW)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, menu.ebo)
	gl.BufferData(
		gl.ELEMENT_ARRAY_BUFFER, glint_size*menu.eboIndexCount, gl.Ptr(menu.eboData), gl.DYNAMIC_DRAW)
	gl.BindVertexArray(0)

	// not necesssary, but i just want to better understand using vertex arrays
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
	return nil
}

func (menu *Menu) ResizeWindow(width float32, height float32) {
	menu.WindowWidth = width
	menu.WindowHeight = height
	menu.Font.ResizeWindow(width, height)
	menu.ortho = mgl32.Ortho2D(-menu.WindowWidth/2, menu.WindowWidth/2, -menu.WindowHeight/2, menu.WindowHeight/2)
}

func (menu *Menu) makeBufferData() {
	// index (0,0)
	menu.vboData[0] = menu.lowerLeft.X // position
	menu.vboData[1] = menu.lowerLeft.Y

	// index (1,0)
	menu.vboData[2] = menu.lowerLeft.X + menu.Width
	menu.vboData[3] = menu.lowerLeft.Y

	// index (1,1)
	menu.vboData[4] = menu.lowerLeft.X + menu.Width
	menu.vboData[5] = menu.lowerLeft.Y + menu.Height

	// index (0,1)
	menu.vboData[6] = menu.lowerLeft.X
	menu.vboData[7] = menu.lowerLeft.Y + menu.Height

	menu.eboData[0] = 0
	menu.eboData[1] = 1
	menu.eboData[2] = 2
	menu.eboData[3] = 0
	menu.eboData[4] = 2
	menu.eboData[5] = 3
}

func (menu *Menu) Release() {
	gl.DeleteBuffers(1, &menu.vbo)
	gl.DeleteBuffers(1, &menu.ebo)
	gl.DeleteBuffers(1, &menu.vao)
}

func (menu *Menu) Draw() bool {
	if !menu.Visible {
		return menu.Visible
	}
	gl.UseProgram(menu.program)

	gl.UniformMatrix4fv(menu.glMatrix, 1, false, &menu.ortho[0])

	gl.BindVertexArray(menu.vao)
	gl.DrawElements(gl.TRIANGLES, int32(menu.eboIndexCount), gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
	for i, label := range menu.Labels {
		if !label.IsHover {
			menu.Labels[i].OnNotHover()
		}
		label.Text.Draw()
	}
	return menu.Visible
}

func (menu *Menu) OrthoToScreenCoord() (x, y float32) {
	x = menu.lowerLeft.X + menu.WindowWidth/2
	y = menu.lowerLeft.Y + menu.WindowHeight/2
	return
}

func (menu *Menu) ScreenClick(xPos, yPos float64) {
	if !menu.Visible {
		return
	}
	yPos = float64(menu.WindowHeight) - yPos
	for i, label := range menu.Labels {
		if label.IsClicked != nil {
			menu.Labels[i].IsClicked(xPos, yPos)
		}
	}
}

func (menu *Menu) ScreenHover(xPos, yPos float64) {
	if !menu.Visible {
		return
	}
	yPos = float64(menu.WindowHeight) - yPos
	for i, label := range menu.Labels {
		if label.IsHovered != nil {
			menu.Labels[i].IsHovered(xPos, yPos)
		}
	}
}

func (menu *Menu) findCenter() (lowerLeft Point) {
	menuWidthHalf := menu.Width / 2
	menuHeightHalf := menu.Height / 2

	lowerLeft.X = -menuWidthHalf
	lowerLeft.Y = -menuHeightHalf
	return
}
