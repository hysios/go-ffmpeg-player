package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/hysios/go-ffmpeg-player/player"
	"github.com/veandco/go-sdl2/sdl"
	// "log"
)

type MethodType int

const (
	MNormal MethodType = iota
	MMap
	MFrame
)

const (
	WINDOWEVENT_RESIZEDEX = sdl.WINDOWEVENT_RESIZED + 100 // 自定义消息
)

var (
	inputfile  string
	outputfile string
	method     int
	// oldWidth  int32 = 1920
	// oldHeight int32 = 1080
	oldWidth  int32 = 800
	oldHeight int32 = 600
	size      string
)

func init() {
	flag.StringVar(&inputfile, "input", "", "a input file, avi, mp4, mkv etc.")
	flag.StringVar(&outputfile, "output", "", "a output file, avi, mp4, mkv etc.")
	flag.IntVar(&method, "method", 0, "0 present normal, 1 present mmap mode")
	flag.StringVar(&size, "scale", "", "set scale size(widthxheight)")

}

func main() {
	flag.Parse()
	var chFrames = make(chan *player.Frame, 100)

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		oldWidth, oldHeight, sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "linear")

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		panic(err)
	}
	// renderer, err := window.GetRenderer()
	// if err != nil {
	// 	panic(err)
	// }
	defer renderer.Destroy()

	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_YV12, sdl.TEXTUREACCESS_STREAMING,
		oldWidth, oldHeight)
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()
	running := true

	renderer.SetLogicalSize(oldWidth, oldHeight)
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()
	renderer.Present()

	// go playing(inputfile, renderer, texture, window)
	go playing(inputfile, chFrames)
	go render(chFrames, texture, renderer, window)
	go Clock()
MainLoop:
	for running {
		for event := sdl.WaitEvent(); event != nil; event = sdl.WaitEvent() {
			switch v := event.(type) {
			case *sdl.WindowEvent:
				if v.Event == WINDOWEVENT_RESIZEDEX {
					window.SetSize(v.Data1, v.Data2)
					// renderer.SetLogicalSize(v.Data1, v.Data2)
					window.SetPosition(sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_UNDEFINED)
				}
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break MainLoop
			}
		}
	}
}

func postWindowChange(frame *player.Frame, texture *sdl.Texture, renderer *sdl.Renderer, window *sdl.Window) (*sdl.Texture, bool) {
	if oldWidth != frame.Width || oldHeight != frame.Height {
		var err error
		oldWidth = frame.Width
		oldHeight = frame.Height
		windowId, _ := window.GetID()

		var event = sdl.WindowEvent{
			Type:     sdl.WINDOWEVENT,
			WindowID: windowId,
			Event:    WINDOWEVENT_RESIZEDEX,
			Data1:    frame.Width,
			Data2:    frame.Height,
		}
		texture.Destroy()

		texture, err = renderer.CreateTexture(sdl.PIXELFORMAT_YV12, sdl.TEXTUREACCESS_TARGET,
			frame.Width, frame.Height)
		if err != nil {
			panic(err)
		}
		// renderer.SetRenderTarget(texture)
		renderer.SetLogicalSize(frame.Width, frame.Height)
		window.SetPosition(sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_UNDEFINED)

		sdl.PushEvent(&event)
		return texture, true
	}
	return texture, false
}

func playing(inputfile string, frames chan<- *player.Frame) {
	ply, _ := player.Open(inputfile,
		&player.Options{Loop: true})
	if len(size) > 0 {
		ply.SetScaleSize(size)
	}
	// ply.SetScale(720, 480)
	ply.Play()

	ply.PreFrame(func(frame *player.Frame) {
		frames <- frame
	})
	ply.Wait()
}

func render(frames <-chan *player.Frame, texture *sdl.Texture, renderer *sdl.Renderer, window *sdl.Window) {
	st := time.Now().UnixNano()
	for frame := range frames {
		if frame == nil {
			continue
		}
		texture, _ = postWindowChange(frame, texture, renderer, window)
		nt := (time.Now().UnixNano() - st) / 1000
		if frame.PTS > 0 {
			time.Sleep(time.Duration(int64(frame.PTS)-nt) * time.Microsecond)
		} else {
			st = time.Now().UnixNano()
		}

		texture.UpdateYUV(nil,
			frame.Data[0], frame.Linesize[0],
			frame.Data[1], frame.Linesize[1],
			frame.Data[2], frame.Linesize[2],
		)

		renderer.Clear()
		renderer.Copy(texture, nil, nil)
		renderer.Present()
	}
}

func Clock() {
	c := time.Tick(time.Second)
	for now := range c {
		fmt.Printf("%v\n", now)
	}
}
