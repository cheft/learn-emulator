package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var font = []uint8{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

type Chip8 struct {
	display [32][64]uint8 // display size
	key     [16]uint8     // input key

	memory [4096]uint8 // memory size 4k
	vx     [16]uint8   // cpu registers V0-VF
	stack  [16]uint16  // program counter stack

	opcode uint16 // current opcode
	i      uint16 // index register
	pc     uint16 // program counter
	sp     uint16 // stack pointer

	delayTimer uint8
	soundTimer uint8
}

func NewChip8() *Chip8 {
	chip8 := &Chip8{
		pc:     0x200,
		opcode: 0,
		i:      0,
		sp:     0,
	}

	for i := 0; i < 80; i++ {
		chip8.memory[i] = font[i]
	}
	return chip8
}

func (c *Chip8) Display() {
	for i := 0; i < len(c.display); i++ {
		for j := 0; j < len(c.display[i]); j++ {
			if c.display[i][j] != 0x0 {
				fmt.Print("■")
			} else {
				fmt.Print("□")
			}
		}
		fmt.Println("")
	}
	// fmt.Println("================================================================")
}

func (c *Chip8) Exec() {
	c.opcode = (uint16(c.memory[c.pc]) << 8) | uint16(c.memory[c.pc+1])
	// fmt.Printf("opcode %X\n", c.opcode)
	// fmt.Printf("opcode %X\n", c.opcode&0xF000)
	switch c.opcode & 0xF000 {
	case 0x0000:
		switch c.opcode & 0x000F {
		case 0x0000: // 0x00E0 Clears screen
			for i := 0; i < len(c.display); i++ {
				for j := 0; j < len(c.display[i]); j++ {
					c.display[i][j] = 0x0
				}
				// TDOO:
				c.pc += 2
				fmt.Println("Clears screen")
			}
		case 0x000E: // 0x00EE Returns from a subroutine
			c.sp -= 1
			c.pc = c.stack[c.sp]
			c.pc += 2
			fmt.Println("Returns from a subroutine")
		default: // 0x0NNN Calls machine code routine (RCA 1802 for COSMAC VIP) at address NNN. Not necessary for most ROMs.
			fmt.Printf("Invalid opcode %X\n", c.opcode)
		}
	case 0x1000: // 0x1NNN Jump to address NNN
		c.pc = c.opcode & 0x0FFF
	case 0x2000: // 0x2NNN Calls subroutine at NNN
		c.stack[c.sp] = c.pc     // store current program counter
		c.sp += 1                // increment stack pointer
		c.pc = c.opcode & 0x0FFF // jump to address NNN
		fmt.Println("Calls subroutine at NNN")
	case 0x3000: // 0x3XNN  Skips the next instruction if VX equals NN.
		if uint16(c.vx[(c.opcode&0x0F00)>>8]) == c.opcode&0x00FF {
			c.pc += 2
		}
		c.pc += 2
	case 0x4000: // 0x4XNN Skips the next instruction if VX doesn't equal NN.
		if uint16(c.vx[(c.opcode&0x0F00)>>8]) != c.opcode&0x00FF {
			c.pc += 2
		}
		c.pc += 2
	case 0x5000: // 0x5XNN Skips the next instruction if VX equals VY.
		if c.vx[(c.opcode&0x0F00)>>8] == c.vx[(c.opcode&0x00F0)>>4] {
			c.pc += 2
		}
		c.pc += 2
	case 0x6000: // 0x6XNN Sets VX to NN
		c.vx[(c.opcode&0x0F00)>>8] = uint8(c.opcode & 0x00FF) // TODO
		c.pc += 2
	case 0x7000: // 0x7XNN Adds NN to VX
		c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] + uint8(c.opcode&0x00FF)
		c.pc += 2
	case 0x8000:
		switch c.opcode & 0x000F {
		case 0x0000: // 0x8XY0 Sets VX to the value of VY.
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x00F0)>>4]
			c.pc += 2
		case 0x0001: // 0x8XY1 Sets VX to VX or VY. (Bitwise OR operation)
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] | c.vx[(c.opcode&0x00F0)>>4]
			c.pc += 2
		case 0x0002: // 0x8XY2 Sets VX to VX and VY. (Bitwise AND operation)
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] & c.vx[(c.opcode&0x00F0)>>4]
			c.pc += 2
		case 0x0003: // 0x8XY3 Sets VX to VX xor VY.
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] ^ c.vx[(c.opcode&0x00F0)>>4]
			c.pc += 2
		case 0x0004: // 0x8XY4 Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't.
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] + c.vx[(c.opcode&0x00F0)>>4]
			if c.vx[(c.opcode&0x0F00)>>8] > 0x00FF {
				c.vx[0xF] = 1
			} else {
				c.vx[0xF] = 0
			}
			c.pc += 2
		case 0x0005: // 0x8XY5 VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if c.vx[(c.opcode&0x00F0)>>4] > c.vx[(c.opcode&0x0F00)>>8] {
				c.vx[0xF] = 0
			} else {
				c.vx[0xF] = 1
			}
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] - c.vx[(c.opcode&0x00F0)>>4]
			c.pc += 2
		case 0x0006: // 0x8XY6 Shifts VY right by one and stores the result to VX (VY remains unchanged). VF is set to the value of the least significant bit of VY before the shift
			c.vx[0xF] = c.vx[(c.opcode&0x0F00)>>8] & 0x1
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] >> 1
			c.pc += 2
		case 0x0007: // 0x8XY7 Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if c.vx[(c.opcode&0x0F00)>>8] > c.vx[(c.opcode&0x00F0)>>4] {
				c.vx[0xF] = 0
			} else {
				c.vx[0xF] = 1
			}
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x00F0)>>4] - c.vx[(c.opcode&0x0F00)>>8]
			c.pc += 2
		case 0x000E: // 0x8XYE Shifts VY left by one and copies the result to VX. VF is set to the value of the most significant bit of VY before the shift
			c.vx[0xF] = c.vx[(c.opcode&0x0F00)>>8] >> 7
			c.vx[(c.opcode&0x0F00)>>8] = c.vx[(c.opcode&0x0F00)>>8] << 1
			c.pc += 2
		}
	case 0x9000: // 0x9XY0 Skips the next instruction if VX doesn't equal VY
		if c.vx[(c.opcode&0x0F00)>>8] != c.vx[(c.opcode&0x00F0)>>4] {
			c.pc += 2
		}
		c.pc += 2
	case 0xA000: // 0xANNN Sets I to the address NNN
		c.i = c.opcode & 0x0FFF
		c.pc += 2
	case 0xB000: // 0xBNNN Jumps to the address NNN plus V0
		c.pc = (c.opcode & 0x0FFF) + uint16(c.vx[0x0])
	case 0xC000: // 0xCXNN Sets VX to the result of a bitwise and operation on a random number (Typically: 0 to 255) and NN
		c.vx[(c.opcode&0x0F00)>>8] = uint8(rand.Intn(256)) & uint8(c.opcode&0x00FF)
		c.pc += 2
	case 0xD000: // 0xDXYN Draws a sprite at coordinate (VX, VY)
		x := c.vx[(c.opcode&0x0F00)>>8]
		y := c.vx[(c.opcode&0x00F0)>>4]
		h := c.opcode & 0x000F
		c.vx[0xF] = 0
		var j uint16 = 0
		var i uint16 = 0
		for j = 0; j < h; j++ {
			pixel := c.memory[c.i+j]
			for i = 0; i < 8; i++ {
				if (pixel & (0x80 >> i)) != 0 {
					if c.display[(y + uint8(j))][x+uint8(i)] == 1 {
						c.vx[0xF] = 1
					}
					c.display[(y + uint8(j))][x+uint8(i)] ^= 1
				}
			}
		} // TODO
		c.pc += 2
	case 0xE000:
		switch c.opcode & 0x00FF {
		case 0x009E: // 0xEX9E Skips the next instruction if the key stored in VX is pressed
			if c.key[c.vx[(c.opcode&0x0F00)>>8]] == 1 {
				c.pc += 2
			}
			c.pc += 2
			// c.pc += 4 // auto press for test
		case 0x00A1: // 0xEXA1 Skips the next instruction if the key stored in VX isn't pressed
			if c.key[c.vx[(c.opcode&0x0F00)>>8]] == 0 {
				c.pc += 2
			}
			c.pc += 2
			// c.pc += 4 // auto press for test
		}
	case 0xF000:
		switch c.opcode & 0x00FF {
		case 0x0007: // 0xFX07 Sets VX to the value of the delay timer
			c.vx[(c.opcode&0x0F00)>>8] = c.delayTimer
			c.pc += 2
		case 0x000A: // 0xFX0A A key press is awaited, and then stored in VX. (Blocking Operation. All instruction halted until next key event)
			pressed := false
			for i := 0; i < len(c.key); i++ {
				if c.key[i] != 0 {
					c.vx[(c.opcode&0x0F00)>>8] = uint8(i)
					pressed = true
				}
			}
			if !pressed {
				fmt.Println("wait key on press")
				return
			}
			c.pc += 2
		case 0x0015: // 0xFX15 Sets the delay timer to VX
			c.delayTimer = c.vx[(c.opcode&0x0F00)>>8]
			c.pc += 2
		case 0x0018: // 0xFX18 Sets the sound timer to VX
			c.soundTimer = c.vx[(c.opcode&0x0F00)>>8]
			c.pc += 2
		case 0x001E: // 0xFX1E Adds VX to I
			if c.i+uint16(c.vx[(c.opcode&0x0F00)>>8]) > 0xFFF {
				c.vx[0xF] = 1
			} else {
				c.vx[0xF] = 0
			}
			c.i = c.i + uint16(c.vx[(c.opcode&0x0F00)>>8])
			c.pc += 2
		case 0x0029: // 0xFX29 Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font
			// TODO
			c.i = uint16(c.vx[(c.opcode&0x0F00)>>8]) * 0x5
			c.pc += 2
		case 0x0033: // 0xFX33 Stores the binary-coded decimal representation of VX, with the most significant of three digits at the address in I, the middle digit at I plus 1, and the least significant digit at I plus 2
			// TODO
			c.memory[c.i] = c.vx[(c.opcode&0x0F00)>>8] / 100          // 百位
			c.memory[c.i+1] = (c.vx[(c.opcode&0x0F00)>>8] / 10) % 10  // 十位
			c.memory[c.i+2] = (c.vx[(c.opcode&0x0F00)>>8] % 100) / 10 // 个位
			c.pc += 2
		case 0x0055: // 0xFX55 Stores V0 to VX (including VX) in memory starting at address I. I is increased by 1 for each value written
			for i := 0; i < int((c.opcode&0x0F00)>>8+1); i++ {
				c.memory[c.i+uint16(i)] = c.vx[i]
			}
			c.i = ((c.opcode & 0x0F00) >> 8) + 1
			c.pc += 2
		case 0x0065: // 0xFX65 Fills V0 to VX (including VX) with values from memory starting at address I. I is increased by 1 for each value written
			for i := 0; i < int((c.opcode&0x0F00)>>8+1); i++ {
				c.vx[i] = c.memory[c.i+uint16(i)]
			}
			c.i = ((c.opcode & 0x0F00) >> 8) + 1
			c.pc += 2
		}
	}
}

func (c *Chip8) Key(num uint8, down bool) {
	if down {
		c.key[num] = 1
	} else {
		c.key[num] = 0
	}
}

func (c *Chip8) LoadRom(filename string) error {
	file, _ := os.OpenFile(filename, os.O_RDONLY, 0777)
	fStat, _ := file.Stat()
	defer file.Close()
	buffer := make([]byte, fStat.Size())
	if _, readErr := file.Read(buffer); readErr != nil {
		return readErr
	}
	for i := 0; i < len(buffer); i++ {
		c.memory[512+i] = buffer[i]
	}
	return nil
}

var chip8 *Chip8

type Game struct{}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		chip8.Key(0x1, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		chip8.Key(0x2, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		chip8.Key(0x3, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.Key4) {
		chip8.Key(0xC, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		chip8.Key(0x4, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		chip8.Key(0x5, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		chip8.Key(0x6, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		chip8.Key(0xD, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		chip8.Key(0x7, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		chip8.Key(0x8, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		chip8.Key(0x9, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		chip8.Key(0xE, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		chip8.Key(0xA, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		chip8.Key(0x0, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		chip8.Key(0xB, true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		chip8.Key(0xF, true)
	}

	if inpututil.IsKeyJustReleased(ebiten.Key1) {
		chip8.Key(0x1, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.Key2) {
		chip8.Key(0x2, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.Key3) {
		chip8.Key(0x3, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.Key4) {
		chip8.Key(0xC, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyQ) {
		chip8.Key(0x4, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyW) {
		chip8.Key(0x5, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyE) {
		chip8.Key(0x6, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyR) {
		chip8.Key(0xD, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyA) {
		chip8.Key(0x7, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyS) {
		chip8.Key(0x8, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyD) {
		chip8.Key(0x9, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyF) {
		chip8.Key(0xE, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyZ) {
		chip8.Key(0xA, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyX) {
		chip8.Key(0x0, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyC) {
		chip8.Key(0xB, false)
	}
	if inpututil.IsKeyJustReleased(ebiten.KeyV) {
		chip8.Key(0xF, false)
	}

	chip8.Exec()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for i := 0; i < len(chip8.display); i++ {
		for j := 0; j < len(chip8.display[i]); j++ {
			if chip8.display[i][j] != 0x0 {
				screen.Set(j+1, i+1, color.White)
			}
		}
	}
	// ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.CurrentTPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 64, 32
}

func main() {
	chip8 = NewChip8()
	// chip8.LoadRom("./roms/pong.c8")
	// chip8.LoadRom("./roms/invaders.c8")
	// chip8.LoadRom("./roms/Zero Demo [zeroZshadow, 2007].ch8")
	chip8.LoadRom("./roms/TETRIS")
	game := &Game{}
	// ebiten.SetWindowSize(64, 32) // 640, 320
	ebiten.SetWindowSize(320, 160) // 640, 320
	ebiten.SetWindowTitle("Chip8")
	// Call ebiten.RunGame to start your game loop.
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
