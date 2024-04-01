package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/profile"
	"github.com/veandco/go-sdl2/sdl"
)

const HEIGHT int32 = 600
const WIDTH int32 = 800

func int_max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}

func int_min(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func crossproduct(x1 int, y1 int, x2 int, y2 int) float32 {
	return (float32(y1) * float32(x2)) - (float32(x1) * float32(y2))
}

type pixel_z struct {
	x int
	y int
	z float32
}

// Colour framebuffer with the zbuffer
func colour_framebuffer_z(framebuffer *[HEIGHT][WIDTH][3]uint8, zbuffer *[HEIGHT][WIDTH]float32, pixels []pixel_z, colour [3]uint8) {
	for _, pixel := range pixels {

		if pixel.x < 0 {
			continue
		}

		if pixel.x >= int(WIDTH) {
			continue
		}

		if pixel.y < 0 {
			continue
		}

		if pixel.y >= int(HEIGHT) {
			continue
		}

		if zbuffer[pixel.y][pixel.x] > pixel.z {
			continue
		}
		framebuffer[pixel.y][pixel.x] = colour
		zbuffer[pixel.y][pixel.x] = pixel.z
	}
}

func generate_triangle(p1 point3d, p2 point3d, p3 point3d) []pixel_z {
	var p1_p2 point3d = point3d{x: p2.x - p1.x, y: p2.y - p1.y, z: p2.z - p1.z}
	var p1_p3 point3d = point3d{x: p3.x - p1.x, y: p3.y - p1.y, z: p3.z - p1.z}

	var normal point3d = point3d{
		x: (p1_p2.y * p1_p3.z) - (p1_p2.z * p1_p3.y),
		y: (p1_p2.z * p1_p3.x) - (p1_p2.x * p1_p3.z),
		z: (p1_p2.x * p1_p3.y) - (p1_p3.y * p1_p3.x),
	}

	var get_z_from_pixel = func(x int, y int) pixel_z {
		// convert it back to screen space
		points := from_coord_space(x, y)
		var z float32 = (normal.x*(points[0]-p1_p2.x)+normal.y*(points[1]-p1_p2.y))/normal.z + p1_p2.z
		return pixel_z{
			x: x,
			y: y,
			z: z,
		}
	}

	coord_p1 := to_coord_space(p1.x, p1.y)
	coord_p2 := to_coord_space(p2.x, p2.y)
	coord_p3 := to_coord_space(p3.x, p3.y)

	points_in_triangle := generate_triangle_barycentric(coord_p1[0], coord_p1[1], coord_p2[0], coord_p2[1], coord_p3[0], coord_p3[1])
	var pixels_z_in_triangle []pixel_z = []pixel_z{}
	for _, point := range points_in_triangle {
		pixels_z_in_triangle = append(pixels_z_in_triangle, get_z_from_pixel(point[0], point[1]))
	}

	return pixels_z_in_triangle
}

// Create a bounding box of the triangle, then use the Barycentric algorithm to decide if points lie in the triangle or not.
func generate_triangle_barycentric(x1 int, y1 int, x2 int, y2 int, x3 int, y3 int) [][2]int {

	if x1 > int(WIDTH) {
		return [][2]int{}
	}

	if x2 > int(WIDTH) {
		return [][2]int{}
	}

	if x3 > int(WIDTH) {
		return [][2]int{}
	}

	if x1 < 0 {
		return [][2]int{}
	}

	if x2 < 0 {
		return [][2]int{}
	}

	if x3 < 0 {
		return [][2]int{}
	}

	if y1 < 0 {
		return [][2]int{}
	}

	if y2 < 0 {
		return [][2]int{}
	}

	if y3 < 0 {
		return [][2]int{}
	}

	if y1 > int(HEIGHT) {
		return [][2]int{}
	}
	if y2 > int(HEIGHT) {
		return [][2]int{}
	}
	if y3 > int(HEIGHT) {
		return [][2]int{}
	}

	var triangle_buffer [][2]int = [][2]int{}

	var max_x int = int_max(x1, int_max(x2, x3))
	var max_y int = int_max(y1, int_max(y2, y3))
	var min_x int = int_min(x1, int_min(x2, x3))
	var min_y int = int_min(y1, int_min(y2, y3))

	var vs1_x int = x2 - x1
	var vs1_y int = y2 - y1

	var vs2_x int = x3 - x1
	var vs2_y int = y3 - y1

	for x := min_x; x < max_x; x++ {
		for y := min_y; y < max_y; y++ {
			var q_x int = x - x1
			var q_y int = y - y1

			var s float32 = crossproduct(q_x, q_y, vs2_x, vs2_y) / crossproduct(vs1_x, vs1_y, vs2_x, vs2_y)
			var t float32 = crossproduct(vs1_x, vs1_y, q_x, q_y) / crossproduct(vs1_x, vs1_y, vs2_x, vs2_y)

			if (s >= 0.0) && (t >= 0.0) && (s+t <= 1) {
				triangle_buffer = append(triangle_buffer, [2]int{x, y})
			}
		}
	}

	return triangle_buffer
}

func clear_buffers(framebuffer *[HEIGHT][WIDTH][3]uint8, zbuffer *[HEIGHT][WIDTH]float32) {
	for x := 0; x < int(WIDTH); x++ {
		for y := 0; y < int(HEIGHT); y++ {
			framebuffer[y][x] = [3]uint8{0, 0, 0}
			zbuffer[y][x] = -math.MaxFloat32
		}
	}
}

func render_screen(surface sdl.Surface, framebuffer *[HEIGHT][WIDTH][3]uint8) {
	for x := 0; x < int(WIDTH); x++ {
		for y := 0; y < int(HEIGHT); y++ {
			surface.Set(x, y, color.RGBA{R: framebuffer[y][x][0], G: framebuffer[y][x][1], B: framebuffer[y][x][2], A: 255})
		}
	}
}

func colour_framebuffer(framebuffer *[HEIGHT][WIDTH][3]uint8, polygon [][2]int, colour [3]uint8) {
	for i := 0; i < len(polygon); i++ {

		if (polygon[i][1] < 0) || (int32(polygon[i][1]) >= HEIGHT) {
			continue
		}

		if (polygon[i][0] < 0) || (int32(polygon[i][0]) >= WIDTH) {
			continue
		}

		framebuffer[polygon[i][1]][polygon[i][0]] = colour
	}
}

type point3d struct {
	x float32
	y float32
	z float32
}

type tri struct {
	p1 point3d
	p2 point3d
	p3 point3d
}

func rotate_point(p point3d, curr_time uint32) point3d {
	var time = float64(curr_time) / 5000

	var angle float64 = (math.Pi * 2.0) * time
	new_x := float32(math.Cos(angle))*p.x + float32(math.Sin(angle))*p.z
	new_z := float32(-math.Sin(angle))*p.x + float32(math.Cos(angle))*p.z
	return point3d{x: new_x, y: p.y, z: new_z}
}

func to_coord_space(x float32, y float32) [2]int {
	var new_x float32 = x*float32(WIDTH)/2 + float32(WIDTH)/2
	var new_y float32 = y*float32(HEIGHT)/2 + float32(HEIGHT)/2

	return [2]int{int(new_x), int(new_y)}
}

func from_coord_space(x int, y int) [2]float32 {
	var new_x float32 = float32(x)/float32(WIDTH)/2.0 - float32(WIDTH)/2.0
	var new_y float32 = float32(y)/float32(HEIGHT)/2 - float32(HEIGHT)/2
	return [2]float32{new_x, new_y}
}

func to_screen_space(p point3d, camera point3d) point3d {

	if p.z+camera.z == 0 {
		return point3d{x: p.x, y: p.y, z: p.z}
	}
	// var x float32 = (p.x*camera.z + camera.x) / (p.z + camera.z)

	var x float32 = (p.x + camera.x) / camera.z
	var y float32 = -1 * (p.y + camera.y) / camera.z
	// var y float32 = -1 * (p.y*camera.z + camera.y) / (p.z + camera.z)

	return point3d{x: x, y: y, z: p.z}
}

func obj_to_triangle(obj string) []tri {
	splitted_obj := strings.Split(obj, "\n")
	var vertices_array []point3d = []point3d{}

	var index int = 0
	for splitted_obj[index][0] != 'f' {
		if splitted_obj[index][0] == 'v' {
			var vertices []string = strings.Split(splitted_obj[index], " ")
			x, err := strconv.ParseFloat(vertices[1], 32)
			if err != nil {
				fmt.Println(x)
			}

			y, err := strconv.ParseFloat(vertices[2], 32)
			if err != nil {
				fmt.Println(y)
			}

			z, err := strconv.ParseFloat(vertices[3], 32)
			if err != nil {
				fmt.Println(z)
			}

			vertices_array = append(vertices_array, point3d{x: float32(x), y: float32(y), z: float32(z)})
		}
		index += 1
	}

	var triangle_array []tri = []tri{}

	fmt.Printf("len of vertices %d\n", len(vertices_array))

	for i := index; i < len(splitted_obj); i++ {
		if splitted_obj[i][0] != 'f' {
			continue
		}

		var vertex_indices []string = strings.Split(splitted_obj[i], " ")

		v1, err := strconv.Atoi(vertex_indices[1])
		if err != nil {
			fmt.Println(v1)
		}

		v2, err := strconv.Atoi(vertex_indices[2])
		if err != nil {
			fmt.Println(v2)
		}

		v3, err := strconv.Atoi(vertex_indices[3])
		if err != nil {
			fmt.Println(v3)
		}

		triangle_array = append(triangle_array, tri{p1: vertices_array[v1-1], p2: vertices_array[v2-1], p3: vertices_array[v3-1]})
	}

	fmt.Printf("len of the triangle array %d\n", len(triangle_array))

	return triangle_array
}

func main() {
	defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		WIDTH, HEIGHT, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	surface, err := window.GetSurface()

	if err != nil {
		panic(err)
	}
	window.UpdateSurface()

	var framebuffer [int(HEIGHT)][int(WIDTH)][3]uint8
	var zbuffer [int(HEIGHT)][int(WIDTH)]float32

	// load in obj file
	dat, err := os.ReadFile("./teapot.obj")

	if err != nil {
		fmt.Println("hello")
	}

	var triangles []tri = obj_to_triangle(string(dat))

	var camera point3d = point3d{x: 0, y: 0, z: 1}

	var fps uint32 = 60
	var desired_delta uint32 = 1000 / fps

	running := true
	for running {
		var init_time uint32 = sdl.GetTicks()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false

			case *sdl.KeyboardEvent:
				camera.z += 0.1
			}
		}

		clear_buffers(&framebuffer, &zbuffer)

		for _, tri := range triangles {
			var p1 point3d = to_screen_space(rotate_point(tri.p1, sdl.GetTicks()), camera)
			var p2 point3d = to_screen_space(rotate_point(tri.p2, sdl.GetTicks()), camera)
			var p3 point3d = to_screen_space(rotate_point(tri.p3, sdl.GetTicks()), camera)

			var colour [3]uint8 = [3]uint8{uint8(rand.IntN(256)), uint8(rand.IntN(256)), uint8(rand.IntN(256))}

			colour_framebuffer_z(&framebuffer, &zbuffer, generate_triangle(p1, p2, p3), colour)
		}

		render_screen(*surface, &framebuffer)

		var delta = sdl.GetTicks() - init_time

		if delta < desired_delta {
			sdl.Delay(desired_delta - delta)
		}
		window.UpdateSurface()
	}
}
