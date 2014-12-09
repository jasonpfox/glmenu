package menu

import (
	gltext "github.com/4ydx/gltext"
	glfw "github.com/go-gl/glfw3"
	"github.com/go-gl/glow/gl-core/3.3/gl"
	"github.com/go-gl/mathgl/mgl32"
	"time"
)

var textboxVertexShader string = `
#version 330

uniform mat4 orthographic_matrix;
uniform vec2 final_position;

in vec4 centered_position;

void main() {
  vec4 center = orthographic_matrix * centered_position;
  gl_Position = vec4(center.x + final_position.x, center.y + final_position.y, center.z, center.w);
}
` + "\x00"

var textboxFragmentShader string = `
#version 330

uniform vec3 background;
out vec4 fragment_color;

void main() {
	fragment_color = vec4(background, 1);
}
` + "\x00"

type TextBoxInteraction func(
	textbox *TextBox,
	xPos, yPos float64,
	button MouseClick,
	isInBoundingBox bool,
)

type TextBox struct {
	Menu               *Menu
	Text               *gltext.Text
	OnClick            TextBoxInteraction
	IsClick            bool
	OnRelease          TextBoxInteraction
	MaxLength          int
	CursorBarFrequency int64
	Time               time.Time
	IsEdit             bool

	// opengl oriented
	program          uint32
	glMatrix         int32
	position         uint32
	vao              uint32
	vbo              uint32
	ebo              uint32
	vboData          []float32
	vboIndexCount    int
	eboData          []int32
	eboIndexCount    int
	centeredPosition uint32

	backgroundUniform         int32
	background                mgl32.Vec3
	finalPositionUniform      int32
	finalPosition             mgl32.Vec2
	orthographicMatrixUniform int32

	// X1, X2: the lower left and upper right points of a box that bounds the text
	X1          Point
	X2          Point
	BorderWidth int32
	Height      int32
	Width       int32
}

func (textbox *TextBox) Load(menu *Menu, font *gltext.Font, width int32, height int32, borderWidth int32) (err error) {
	textbox.Menu = menu

	// text
	textbox.CursorBarFrequency = time.Duration.Nanoseconds(500000000)
	textbox.Text = gltext.LoadText(font)

	// border formatting
	textbox.BorderWidth = borderWidth
	textbox.Height = height
	textbox.Width = width
	textbox.X1.X = -float32(width) / 2.0
	textbox.X1.Y = -float32(height) / 2.0
	textbox.X2.X = float32(width) / 2.0
	textbox.X2.Y = float32(height) / 2.0
	textbox.background = mgl32.Vec3{1.0, 1.0, 1.0}

	// create shader program and define attributes and uniforms
	textbox.program, err = gltext.NewProgram(textboxVertexShader, textboxFragmentShader)
	if err != nil {
		return err
	}

	// ebo, vbo data
	// 4 edges with 4 vertices apiece
	//	textbox.vboIndexCount = 16 * 2 // 2 position points per index
	textbox.vboIndexCount = 8 * 2 // 2 position points per index
	textbox.eboIndexCount = 12
	textbox.vboData = make([]float32, textbox.vboIndexCount, textbox.vboIndexCount)
	textbox.eboData = make([]int32, textbox.eboIndexCount, textbox.eboIndexCount)
	textbox.makeBufferData()

	// attributes
	textbox.centeredPosition = uint32(gl.GetAttribLocation(textbox.program, gl.Str("centered_position\x00")))

	// uniforms
	textbox.backgroundUniform = gl.GetUniformLocation(textbox.program, gl.Str("background\x00"))
	textbox.finalPositionUniform = gl.GetUniformLocation(textbox.program, gl.Str("final_position\x00"))
	textbox.orthographicMatrixUniform = gl.GetUniformLocation(textbox.program, gl.Str("orthographic_matrix\x00"))

	gl.GenVertexArrays(1, &textbox.vao)
	gl.GenBuffers(1, &textbox.vbo)
	gl.GenBuffers(1, &textbox.ebo)

	glfloatSize := int32(4)

	// vao
	gl.BindVertexArray(textbox.vao)

	// vbo
	gl.BindBuffer(gl.ARRAY_BUFFER, textbox.vbo)

	gl.EnableVertexAttribArray(textbox.centeredPosition)
	gl.VertexAttribPointer(
		textbox.centeredPosition,
		2,
		gl.FLOAT,
		false,
		0,
		gl.PtrOffset(0),
	)
	gl.BufferData(gl.ARRAY_BUFFER, int(glfloatSize)*textbox.vboIndexCount, gl.Ptr(textbox.vboData), gl.DYNAMIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, textbox.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(glfloatSize)*textbox.eboIndexCount, gl.Ptr(textbox.eboData), gl.DYNAMIC_DRAW)
	gl.BindVertexArray(0)

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
	return
}

func (textbox *TextBox) makeBufferData() {
	// left edge - positions starting at upper right CCW
	// this all works because the original positioning is centered around the origin
	textbox.vboData[0] = textbox.X1.X
	textbox.vboData[1] = textbox.X1.Y + float32(textbox.Height) + float32(textbox.BorderWidth)

	textbox.vboData[2] = textbox.X1.X - float32(textbox.BorderWidth)
	textbox.vboData[3] = textbox.X1.Y + float32(textbox.Height) + float32(textbox.BorderWidth)

	textbox.vboData[4] = textbox.X1.X - float32(textbox.BorderWidth)
	textbox.vboData[5] = textbox.X1.Y - float32(textbox.BorderWidth)

	textbox.vboData[6] = textbox.X1.X
	textbox.vboData[7] = textbox.X1.Y - float32(textbox.BorderWidth)

	// top edge - intentionally leaves out the borderwidth on the x-axis
	textbox.vboData[8] = textbox.X2.X
	textbox.vboData[9] = textbox.X2.Y + float32(textbox.BorderWidth)

	textbox.vboData[10] = textbox.X2.X - float32(textbox.Width)
	textbox.vboData[11] = textbox.X2.Y + float32(textbox.BorderWidth)

	textbox.vboData[12] = textbox.X2.X - float32(textbox.Width)
	textbox.vboData[13] = textbox.X2.Y

	textbox.vboData[14] = textbox.X2.X
	textbox.vboData[15] = textbox.X2.Y

	textbox.eboData[0] = 0
	textbox.eboData[1] = 1
	textbox.eboData[2] = 2
	textbox.eboData[3] = 0
	textbox.eboData[4] = 2
	textbox.eboData[5] = 3

	textbox.eboData[6] = 4
	textbox.eboData[7] = 5
	textbox.eboData[8] = 6
	textbox.eboData[9] = 4
	textbox.eboData[10] = 6
	textbox.eboData[11] = 7
}

func (textbox *TextBox) SetString(str string, argv ...interface{}) {
	if len(argv) == 0 {
		textbox.Text.SetString(str + "|")
	} else {
		textbox.Text.SetString(str+"|", argv)
	}
}

func (textbox *TextBox) Draw() {
	if time.Since(textbox.Time).Nanoseconds() > textbox.CursorBarFrequency {
		if textbox.Text.RuneCount < textbox.Text.GetLength() {
			textbox.Text.RuneCount = textbox.Text.GetLength()
		} else {
			textbox.Text.RuneCount -= 1
		}
		textbox.Time = time.Now()
	}
	if !textbox.IsEdit {
		// dont show flashing bar unless actually editing
		textbox.Text.RuneCount = textbox.Text.GetLength() - 1
	}
	gl.Disable(gl.BLEND)
	gl.UseProgram(textbox.program)

	// uniforms
	gl.Uniform3fv(textbox.backgroundUniform, 1, &textbox.background[0])
	gl.Uniform2fv(textbox.finalPositionUniform, 1, &textbox.finalPosition[0])
	gl.UniformMatrix4fv(textbox.orthographicMatrixUniform, 1, false, &textbox.Menu.Font.OrthographicMatrix[0])

	// draw
	gl.BindVertexArray(textbox.vao)
	gl.DrawElements(gl.TRIANGLES, int32(textbox.eboIndexCount), gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)

	textbox.Text.Draw()
}

func (textbox *TextBox) KeyPress(key glfw.Key, withShift bool) {
	switch key {
	case glfw.KeyBackspace:
		textbox.Backspace()
	default:
		textbox.AddCharacter(key, withShift)
	}
}

func (textbox *TextBox) AddCharacter(key glfw.Key, withShift bool) {
	if textbox.Text.HasRune(rune(key)) {
		var theRune rune
		if !withShift && key >= 65 && key <= 90 {
			theRune = rune(key) + 32
		} else {
			theRune = rune(key)
		}
		r := []rune(textbox.Text.String)
		r = r[0 : len(r)-1] // trim the bar
		r = append(r, theRune)
		textbox.Text.SetString(string(r) + "|")
		textbox.Text.SetPosition(textbox.Text.SetPositionX, textbox.Text.SetPositionY)
	}
}

func (textbox *TextBox) Backspace() {
	if textbox.IsEdit {
		r := []rune(textbox.Text.String)
		if len(r) > 1 {
			r = r[0 : len(r)-2]
			// this will recenter the textbox on the screen
			textbox.Text.SetString(string(r) + "|")
			// this will place it back where it was previously positioned
			textbox.Text.SetPosition(textbox.Text.SetPositionX, textbox.Text.SetPositionY)
		}
	}
}

func (textbox *TextBox) OrthoToScreenCoord() (X1 Point, X2 Point) {
	X1.X = textbox.Text.X1.X + textbox.Menu.WindowWidth/2
	X1.Y = textbox.Text.X1.Y + textbox.Menu.WindowHeight/2

	X2.X = textbox.Text.X2.X + textbox.Menu.WindowWidth/2
	X2.Y = textbox.Text.X2.Y + textbox.Menu.WindowHeight/2
	return
}

func (textbox *TextBox) IsClicked(xPos, yPos float64, button MouseClick) {
	// menu rendering (and text) is positioned in orthographic projection coordinates
	// but click positions are based on window coordinates
	// we have to transform them
	X1, X2 := textbox.OrthoToScreenCoord()
	inBox := float32(xPos) > X1.X && float32(xPos) < X2.X && float32(yPos) > X1.Y && float32(yPos) < X2.Y
	if inBox {
		textbox.IsClick = true
		if textbox.OnClick != nil {
			textbox.OnClick(textbox, xPos, yPos, button, inBox)
		}
	} else {
		textbox.IsEdit = false
	}
}

func (textbox *TextBox) IsReleased(xPos, yPos float64, button MouseClick) {
	// anything flagged as clicked now needs to decide whether to execute its logic based on inBox
	X1, X2 := textbox.OrthoToScreenCoord()
	inBox := float32(xPos) > X1.X && float32(xPos) < X2.X && float32(yPos) > X1.Y && float32(yPos) < X2.Y
	if textbox.IsClick {
		textbox.IsEdit = true
		if textbox.OnRelease != nil {
			textbox.OnRelease(textbox, xPos, yPos, button, inBox)
		}
	}
	textbox.IsClick = false
}
